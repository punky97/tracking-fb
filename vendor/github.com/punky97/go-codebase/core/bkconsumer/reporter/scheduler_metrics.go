package reporter

import (
	"encoding/json"
	"github.com/punky97/go-codebase/core/bkconsumer/utils"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"github.com/punky97/go-codebase/core/utils/sync"
	"github.com/spf13/viper"
	"os"
	"sync/atomic"
	"time"
)

type handlerMetrics struct {
	ProcessRuntime int64
	CallCounter    int64
	ResponseCount  *sync.IntIntMap
}

type schedulerMetrics struct {
	producers       []*queue.Producer
	handlerMetrics  *sync.HashMap
	lastOutputCount int64
}

type SchedulerWorkerMetrics struct {
	closed      chan struct{}
	startReport chan struct{}
	Uptime      time.Time
	Pid         int
	appName     string
	podName     string

	interval time.Duration
	metrics  *schedulerMetrics
}

func NewSchedulerWorkerMetrics(appName string) *SchedulerWorkerMetrics {
	intervalTime := viper.GetInt64("reporter.interval")
	if intervalTime <= 0 {
		intervalTime = constant.DefaultReportTime
	}

	workerMetrics := &SchedulerWorkerMetrics{
		closed:      make(chan struct{}, 1),
		startReport: make(chan struct{}, 1),
		Uptime:      time.Now(),
		Pid:         os.Getpid(),
		appName:     appName,
		podName:     config.GetHostName(),

		interval: time.Duration(intervalTime) * time.Millisecond,
		metrics: &schedulerMetrics{
			producers:      make([]*queue.Producer, 0),
			handlerMetrics: sync.NewHashMap(),
		},
	}

	if len(appName) == 0 {
		return workerMetrics
	}

	if !reporter.IsMetricsEnable() {
		return workerMetrics
	}

	if reporter.IsAllMetricsEnable() {
		return workerMetrics
	}

	reporter.GetProducerFromConf()
	go func() {
		select {
		case <-workerMetrics.closed:
			return
		case <-workerMetrics.startReport:
			break
		}

		ticker := time.NewTicker(workerMetrics.interval)
		defer ticker.Stop()

		for {
			select {
			case <-workerMetrics.closed:
				data, _ := workerMetrics.metrics.Metrics()
				data.App = workerMetrics.appName
				data.PodName = workerMetrics.podName
				if data.ExecuteCount > 0 {
					dataBytes, err := json.Marshal(data)
					if err == nil {
						logger.BkLog.Infof("Metrics: %v", string(dataBytes))
						reporter.PublishSimple("core_app_metrics", dataBytes)
					}
				}

				return
			case <-ticker.C:
				data, output := workerMetrics.metrics.Metrics()
				data.App = workerMetrics.appName
				data.PodName = workerMetrics.podName
				if data.ExecuteCount > 0 {
					dataBytes, err := json.Marshal(data)
					if err == nil {
						logger.BkLog.Infof("Metrics: %v", string(dataBytes))
						reporter.PublishSimple("core_app_metrics", dataBytes)
					}

					workerMetrics.metrics.ResetMetricsWitOutputCount(output)
				}
			}
		}
	}()

	return workerMetrics
}

// SchedulerWorkerMetrics func

func (m *SchedulerWorkerMetrics) Close() {
	close(m.closed)
}

func (m *SchedulerWorkerMetrics) Start() {
	close(m.startReport)
}

func (m *SchedulerWorkerMetrics) AddWatchingProducer(p *queue.Producer) {
	m.metrics.producers = append(m.metrics.producers, p)
}

// schedulerMetrics func

func (m *schedulerMetrics) GetOrCreateHandlerMetrics(name string) *handlerMetrics {
	return m.handlerMetrics.GetOrInsert(name, NewHandlerMetrics()).(*handlerMetrics)
}

func NewHandlerMetrics() *handlerMetrics {
	return &handlerMetrics{
		ProcessRuntime: 0,
		CallCounter:    0,
		ResponseCount:  sync.NewIntIntMap(),
	}
}

func (h *handlerMetrics) End(d time.Duration) {
	// only encounter when job is done
	atomic.AddInt64(&(h.CallCounter), 1)
	atomic.AddInt64(&(h.ProcessRuntime), d.Nanoseconds())
}

func (h *handlerMetrics) Handled(code int) {
	h.ResponseCount.Increase(int64(code), 1)
}

func (m *schedulerMetrics) ResetMetricsWitOutputCount(outputCount int64) {
	var keys = make([]string, 0)
	for kv := range m.handlerMetrics.Iter() {
		keys = append(keys, kv.Key.(string))
	}

	for _, k := range keys {
		m.handlerMetrics.Set(k, NewHandlerMetrics())
	}

	m.lastOutputCount = outputCount
}

func (m *schedulerMetrics) Metrics() (*metrics.SchedulerMetrics, int64) {
	hms := make(map[string]metrics.SchedulerHandlerMetrics)

	processTime := int64(0)
	callCounter := int64(0)
	totalResponseCount := int64(0)
	totalErrorCount := int64(0)
	for kv := range m.handlerMetrics.Iter() {
		var name = kv.Key.(string)
		var mt = kv.Value.(*handlerMetrics)
		var callCount = atomic.LoadInt64(&(mt.CallCounter))
		if callCount > 0 {
			duration := atomic.LoadInt64(&(mt.ProcessRuntime))

			errorCount := int64(0)
			responseCount := int64(0)
			sttCountMap := mt.ResponseCount.ToMap()
			for code, count := range sttCountMap {
				responseCount += count
				if IsSchedulerErrorCode(int(code)) {
					errorCount += count
				}
			}

			hms[name] = metrics.SchedulerHandlerMetrics{
				AverageRuntime: time.Duration(duration).Seconds() / float64(callCount),
				ExecuteCount:   callCount,
				FailRate:       float64(errorCount) / float64(callCount),
				FailCounter:    errorCount,
			}

			processTime += duration
			callCounter += callCount
			totalResponseCount += responseCount
			totalErrorCount += errorCount
		}
	}

	totalOutputCount := int64(0)
	for _, producer := range m.producers {
		totalOutputCount += producer.OutputCount()
	}
	outputCount := totalOutputCount - m.lastOutputCount

	errorRate := float64(0)
	averageTime := float64(0)
	if callCounter > 0 {
		errorRate = float64(totalErrorCount) / float64(callCounter)
		averageTime = time.Duration(processTime / callCounter).Seconds()
	}

	// do not dump mt until scheduler is done
	if totalResponseCount == 0 {
		callCounter = 0
	}

	r := &metrics.SchedulerMetrics{
		ExecuteCount:   callCounter,
		AverageRuntime: averageTime,
		FailCounter:    totalErrorCount,
		FailRate:       errorRate,
		OutputCount:    outputCount,
		HandlerMetrics: hms,
	}

	r.Label = "scheduler_metrics"
	r.ProcStats = reporter.GetProcStats()
	r.Timestamp = time.Now().Unix()
	r.EnableMetrics = "all"

	return r, totalOutputCount
}

func IsSchedulerErrorCode(code int) bool {
	return code != int(utils.ProcessOKCode)
}
