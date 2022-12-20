package reporter

import (
	"encoding/json"
	"fmt"
	"github.com/punky97/go-codebase/core/bkconsumer/utils"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/utils/sync"
	"github.com/spf13/viper"
	"os"
	"sync/atomic"
	"time"
)

const (
	MaxErrorMessageStackSize = 20
)

var (
	enableAllConsumerMetrics = false
)

type queueMessage struct {
	Message string `json:"message"`
	Task    string `json:"task"`
}

type taskMetrics struct {
	statusCounter   *sync.IntIntMap
	consumerCounter int64
	errorCounter    int64
	totalTime       int64
	minTime         int64
	maxTime         int64
	errorMsgStack   *sync.HashMap
}

type consumerMetrics struct {
	tasks *sync.HashMap
}

type ConsumerWorkerMetrics struct {
	closed      chan struct{}
	startReport chan struct{}
	Uptime      time.Time
	Pid         int
	appName     string
	podName     string

	interval time.Duration
	metrics  *consumerMetrics
}

func NewConsumerWorkerMetrics(appName string) *ConsumerWorkerMetrics {
	intervalTime := viper.GetInt64("reporter.interval")
	if intervalTime <= 0 {
		intervalTime = constant.DefaultReportTime
	}

	workerMetrics := &ConsumerWorkerMetrics{
		closed:      make(chan struct{}, 1),
		startReport: make(chan struct{}, 1),
		Uptime:      time.Now(),
		Pid:         os.Getpid(),
		appName:     appName,
		podName:     config.GetHostName(),

		interval: time.Duration(intervalTime) * time.Millisecond,
		metrics: &consumerMetrics{
			tasks: sync.NewHashMap(),
		},
	}

	if len(appName) == 0 {
		return workerMetrics
	}

	if !reporter.IsMetricsEnable() {
		return workerMetrics
	}

	enableAllConsumerMetrics = reporter.IsAllMetricsEnable()

	slackNoti := notifier.DefaultNotifier()
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

		var lastReportTime = time.Now().Unix()
		for {
			select {
			case <-workerMetrics.closed:
				data, errorMsgs := workerMetrics.metrics.Metrics(time.Since(time.Unix(lastReportTime, 0)))
				data.App = workerMetrics.appName
				data.PodName = workerMetrics.podName
				dataBytes, _ := json.Marshal(data)
				logger.BkLog.Infof("Metrics: %v", string(dataBytes))
				reporter.PublishSimple("core_app_metrics", dataBytes)

				if len(errorMsgs) > 0 {
					slackNoti.Notify(slackNoti.Format(constant.ReportLog{
						ReportType: constant.ConsumerErrorReport,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"service":  appName,
							"source":   workerMetrics.podName,
							"messages": errorMsgs,
						},
					}))
					logger.BkLog.Warnf("Alert Consumer: %v", errorMsgs)
				}

				return
			case <-ticker.C:
				data, errorMsgs := workerMetrics.metrics.Metrics(workerMetrics.interval)
				data.App = workerMetrics.appName
				data.PodName = workerMetrics.podName
				dataBytes, _ := json.Marshal(data)
				logger.BkLog.Infof("Metrics: %v", string(dataBytes))
				reporter.PublishSimple("core_app_metrics", dataBytes)

				if len(errorMsgs) > 0 {
					// TODO: notify to slack
					logger.BkLog.Warnf("Alert Consumer: %v", errorMsgs)
				}

				workerMetrics.metrics.ResetMetrics()
				lastReportTime = time.Now().Unix()
			}
		}
	}()

	return workerMetrics
}

// ConsumerWorkerMetrics func

func (m *ConsumerWorkerMetrics) Close() {
	close(m.closed)
}

func (m *ConsumerWorkerMetrics) Start() {
	close(m.startReport)
}

// ConsumerTaskMetrics func

func (m *consumerMetrics) GetOrCreateTaskMetrics(consumerName string) *taskMetrics {
	return m.tasks.GetOrInsert(consumerName, NewTaskMetrics()).(*taskMetrics)
}

func NewTaskMetrics() *taskMetrics {
	return &taskMetrics{
		statusCounter:   sync.NewIntIntMap(),
		consumerCounter: 0,
		errorCounter:    0,
		totalTime:       0,
		minTime:         0,
		maxTime:         0,
		errorMsgStack:   sync.NewHashMap(),
	}
}

func (t *taskMetrics) End(d time.Duration) {
	atomic.AddInt64(&(t.consumerCounter), 1)

	if enableAllConsumerMetrics {
		dNs := d.Nanoseconds()
		minTime := atomic.LoadInt64(&(t.minTime))
		maxTime := atomic.LoadInt64(&(t.maxTime))

		atomic.AddInt64(&(t.totalTime), dNs)

		if dNs < minTime {
			atomic.StoreInt64(&(t.minTime), dNs)
		}
		if dNs > maxTime {
			atomic.StoreInt64(&(t.maxTime), dNs)
		}
	}
}

func (t *taskMetrics) Handled(code int) {
	if IsConsumerErrorCode(code) {
		atomic.AddInt64(&(t.errorCounter), 1)
	}

	if enableAllConsumerMetrics {
		t.statusCounter.Increase(int64(code), 1)
	}
}

func (t *taskMetrics) Report(consumerName string, reportMsg []byte) {
	if t.errorMsgStack.Len() < MaxErrorMessageStackSize {
		t.errorMsgStack.Set(fmt.Sprintf("%d", t.errorMsgStack.Len()), &queueMessage{string(reportMsg), consumerName})
	}
}

func (m *consumerMetrics) ResetMetrics() {
	var keys = make([]string, 0)
	for kv := range m.tasks.Iter() {
		keys = append(keys, kv.Key.(string))
	}

	for _, k := range keys {
		m.tasks.Set(k, NewTaskMetrics())
	}
}

func (m *consumerMetrics) Metrics(interval time.Duration) (*metrics.ConsumerMetrics, []queueMessage) {
	tasks := make(map[string]metrics.ConsumerTaskMetrics)
	errorMsgs := make([]queueMessage, 0)

	totalConsumeCount := int64(0)
	totalErrorCount := int64(0)
	totalTime := int64(0)
	for kv := range m.tasks.Iter() {
		var name = kv.Key.(string)
		var task = kv.Value.(*taskMetrics)
		var consumeCount = atomic.LoadInt64(&task.consumerCounter)
		if consumeCount > 0 {
			var duration int64
			var errorCount = atomic.LoadInt64(&(task.errorCounter))

			var taskMt = metrics.ConsumerTaskMetrics{
				ErrorRate:    float64(errorCount) / float64(consumeCount),
				ErrorCounter: errorCount,
			}

			if interval > 0 {
				taskMt.ConsumerRate = float64(consumeCount) / interval.Seconds() * 60.0
			}

			if enableAllConsumerMetrics {
				duration = atomic.LoadInt64(&(task.totalTime))

				minTime := atomic.LoadInt64(&task.minTime)
				maxTime := atomic.LoadInt64(&task.maxTime)

				taskMt.AverageProcessTime = time.Duration(duration).Seconds() / float64(consumeCount)
				taskMt.MinTime = time.Duration(minTime).Seconds()
				taskMt.MaxTime = time.Duration(maxTime).Seconds()
				taskMt.StatusCounter = task.statusCounter.ToMap()
			}

			tasks[name] = taskMt

			totalConsumeCount += consumeCount
			totalErrorCount += errorCount
			totalTime += duration

			// push error message stack
			for kv := range task.errorMsgStack.Iter() {
				errorMsgs = append(errorMsgs, *(kv.Value.(*queueMessage)))
			}
		}
	}

	consumeRate := 0.0
	if interval > 0 {
		consumeRate = float64(totalConsumeCount) / interval.Seconds()
	}

	errorRate := 0.0
	avgTime := 0.0
	if totalConsumeCount > 0 {
		errorRate = float64(totalErrorCount) / float64(totalConsumeCount)
		avgTime = time.Duration(totalTime / totalConsumeCount).Seconds()
	}

	r := &metrics.ConsumerMetrics{
		AverageProcessTime: avgTime,
		ConsumeRate:        consumeRate,
		ErrorRate:          errorRate,
		ErrorCounter:       totalErrorCount,
		Tasks:              tasks,
	}

	r.Label = "consumer_metrics"
	r.ProcStats = reporter.GetProcStats()
	r.Timestamp = time.Now().Unix()
	if enableAllConsumerMetrics {
		r.EnableMetrics = "all"
	} else {
		r.EnableMetrics = "error_count,error_rate,consume_rate"
	}

	return r, errorMsgs
}

func IsConsumerErrorCode(code int) bool {
	return code != int(utils.ProcessOKCode)
}
