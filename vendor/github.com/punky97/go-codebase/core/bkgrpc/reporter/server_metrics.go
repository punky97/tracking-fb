package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/punky97/go-codebase/core/bkgrpc/common"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/utils/sync"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"math"
	"os"
	"reflect"
	"sync/atomic"
	"time"
)

var (
	enableAllDMSMetrics = false
)

type EndpointMetrics struct {
	MaxTime     int64
	MinTime     int64
	Count       int64
	ErrorCount  int64
	Sent        int64
	Receive     int64
	SentSize    int64
	ReceiveSize int64
	StatusCount *sync.IntHashMap
	TotalTime   int64
}

type dmsMetrics struct {
	serverResponseTimes *sync.HashMap // endpoint metrics
}

type ServerMetrics struct {
	closed      chan struct{}
	startReport chan struct{}
	Uptime      time.Time
	Pid         int
	appName     string
	podName     string

	reportMetrics  *dmsMetrics
	reportInterval time.Duration

	alertMetrics  *dmsMetrics
	alertInterval time.Duration
}

func NewServerMetrics(serviceName string) *ServerMetrics {
	intervalTime := viper.GetInt64("reporter.interval")
	if intervalTime <= 0 {
		intervalTime = constant.DefaultReportTime
	}

	alertIntervalTime := viper.GetInt64("reporter.alert_interval")
	if alertIntervalTime <= 0 {
		alertIntervalTime = constant.DefaultAlertTime
	}

	appPodName := config.GetHostName()
	serverMetrics := &ServerMetrics{
		closed:      make(chan struct{}, 1),
		startReport: make(chan struct{}, 1),
		Uptime:      time.Now(),
		Pid:         os.Getpid(),
		appName:     serviceName,
		podName:     appPodName,

		reportMetrics: &dmsMetrics{
			serverResponseTimes: sync.NewHashMap(),
		},
		reportInterval: time.Duration(intervalTime) * time.Millisecond,

		alertMetrics: &dmsMetrics{
			serverResponseTimes: sync.NewHashMap(),
		},
		alertInterval: time.Duration(alertIntervalTime) * time.Millisecond,
	}

	if !reporter.IsMetricsEnable() {
		return serverMetrics
	}

	enableAllDMSMetrics = reporter.IsAllMetricsEnable()

	alertConfigs := reporter.GetAPIDMSAlertConfig(serviceName)
	reporter.GetProducerFromConf()
	slackNoti := notifier.DefaultNotifier()

	// report metric routine
	go func() {
		select {
		case <-serverMetrics.closed:
			return
		case <-serverMetrics.startReport:
			break
		}

		ticker := time.NewTicker(serverMetrics.reportInterval)
		defer ticker.Stop()

		lastReportTime := time.Now().Unix()
		for {
			select {
			case <-serverMetrics.closed:
				data := serverMetrics.GetReportMetrics(time.Since(time.Unix(lastReportTime, 0)))
				if data.RequestPerMin > 0 {
					dataBytes, _ := json.Marshal(data)
					logger.BkLog.Infof("Metrics: %v", string(dataBytes))
					reporter.PublishSimple("core_app_metrics", dataBytes)
				}

				return
			case <-ticker.C:
				data := serverMetrics.GetReportMetrics(serverMetrics.reportInterval)
				dataBytes, _ := json.Marshal(data)
				logger.BkLog.Infof("Metrics: %v", string(dataBytes))
				reporter.PublishSimple("core_app_metrics", dataBytes)

				serverMetrics.reportMetrics.ResetIntervalCounts()
				lastReportTime = time.Now().Unix()
			}
		}
	}()

	// alert metrics routine
	go func() {
		select {
		case <-serverMetrics.closed:
			return
		case <-serverMetrics.startReport:
			break
		}

		ticker := time.NewTicker(serverMetrics.alertInterval)
		defer ticker.Stop()

		lastAlertTime := time.Now().Unix()
		for {
			select {
			case <-serverMetrics.closed:
				data := serverMetrics.GetAlertMetrics(time.Since(time.Unix(lastAlertTime, 0)))
				endpoints := make(map[string]interface{})
				for endpoint, entry := range data.EndpointResponseTimes {
					for _, conf := range alertConfigs {
						if entry.ErrorRate >= conf.ErrorRate &&
							entry.RPM >= float64(conf.RequestFrom) &&
							entry.RPM <= float64(conf.RequestTo) {
							endpoints[endpoint] = entry
							break
						}
					}
				}

				if len(endpoints) > 0 {
					slackNoti.Notify(slackNoti.Format(constant.ReportLog{
						ReportType: constant.DMSHealthReport,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"service":   serviceName,
							"source":    appPodName,
							"endpoints": endpoints,
						},
					}))
					dataBytes, _ := json.Marshal(endpoints)
					logger.BkLog.Infof("Alert DMS %v: %v", serviceName, string(dataBytes))
				}

				return
			case <-ticker.C:
				data := serverMetrics.GetAlertMetrics(serverMetrics.alertInterval)
				endpoints := make(map[string]interface{})
				for endpoint, entry := range data.EndpointResponseTimes {
					for _, conf := range alertConfigs {
						if entry.ErrorRate >= conf.ErrorRate &&
							entry.RPM >= float64(conf.RequestFrom) &&
							entry.RPM <= float64(conf.RequestTo) {
							endpoints[endpoint] = entry
							break
						}
					}
				}

				if len(endpoints) > 0 {
					slackNoti.Notify(slackNoti.Format(constant.ReportLog{
						ReportType: constant.DMSHealthReport,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"service":   serviceName,
							"source":    appPodName,
							"endpoints": endpoints,
						},
					}))

					dataBytes, _ := json.Marshal(endpoints)
					logger.BkLog.Infof("Alert DMS %v: %v", serviceName, string(dataBytes))
				}

				serverMetrics.alertMetrics.ResetIntervalCounts()
				lastAlertTime = time.Now().Unix()
			}
		}
	}()

	return serverMetrics
}

func (m *ServerMetrics) Close() {
	close(m.closed)
}

func (m *ServerMetrics) Start() {
	close(m.startReport)
}

// UnaryServerInterceptor is a gRPC server-side interceptor that provides Prometheus monitoring for Unary RPCs.
func (m *ServerMetrics) UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		defer func() {
			if perr := recover(); perr != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Panic on server metrics: %v", perr))
			}
		}()

		recvSize := proto.Size(req.(proto.Message))
		monitor := newServerReporter(m, Unary, info.FullMethod)

		resp, err := handler(ctx, req)

		// convert DeadlineExceeded and Cancel context to error
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			resp = reflect.New(reflect.TypeOf(resp).Elem()).Interface()

			if ctx.Err() == context.Canceled {
				err = status.Error(codes.Canceled, ctx.Err().Error())
			} else if ctx.Err() == context.DeadlineExceeded {
				err = status.Error(codes.DeadlineExceeded, ctx.Err().Error())
			}
		}

		if enableAllDMSMetrics {
			monitor.ReceivedMessage(recvSize)
		}

		code := common.GrpcErrorConvert(err)
		monitor.Handled(code)
		if !common.IsErrorCode(code.String()) {
			if enableAllDMSMetrics {
				sentSize := proto.Size(resp.(proto.Message))
				monitor.SentMessage(sentSize)
			}
		} else {
			p, _ := peer.FromContext(ctx)
			if p != nil {
				sentSize := 0
				if re, ok := resp.(proto.Message); ok {
					sentSize = proto.Size(re)
				}
				logger.BkLog.Infof("Req err from: %v, method: %v, size: %v, code: %v, err: %v", p.Addr, info.FullMethod, sentSize, code, err)
			}
		}

		return resp, err
	}
}
func (m *ServerMetrics) GetReportMetrics(duration time.Duration) *metrics.DMSMetrics {
	mt := m.reportMetrics.Metrics(duration)
	mt.App = m.appName
	mt.PodName = m.podName

	return mt
}

func (m *ServerMetrics) GetAlertMetrics(duration time.Duration) *metrics.DMSMetrics {
	mt := m.alertMetrics.Metrics(duration)
	mt.App = m.appName
	mt.PodName = m.podName

	return mt
}

func (e *EndpointMetrics) Call() {
	atomic.AddInt64(&(e.Count), 1)
}

func (e *EndpointMetrics) ReceivedMessage(size int) {
	atomic.AddInt64(&(e.Receive), 1)
	atomic.AddInt64(&(e.ReceiveSize), int64(size))
}

func (e *EndpointMetrics) SentMessage(size int, startTime time.Time) {
	atomic.AddInt64(&(e.Sent), 1)

	handleTime := time.Since(startTime).Nanoseconds()
	minTime := atomic.LoadInt64(&(e.MinTime))
	maxTime := atomic.LoadInt64(&(e.MaxTime))
	if minTime > handleTime {
		atomic.StoreInt64(&(e.MinTime), handleTime)
	}
	if maxTime < handleTime {
		atomic.StoreInt64(&(e.MaxTime), handleTime)
	}

	atomic.AddInt64(&(e.SentSize), int64(size))
	atomic.AddInt64(&(e.TotalTime), handleTime)
}

func (e *EndpointMetrics) Handled(code codes.Code) {
	if common.IsErrorCode(code.String()) {
		atomic.AddInt64(&(e.ErrorCount), 1)
	}

	if enableAllDMSMetrics {
		e.StatusCount.Increase(code.String(), 1)
	}
}

func (m *dmsMetrics) ResetIntervalCounts() {
	var keys = make([]string, 0)
	for k := range m.serverResponseTimes.Iter() {
		keys = append(keys, k.Key.(string))
	}

	for _, k := range keys {
		m.serverResponseTimes.Set(k, NewEndpointMetrics())
	}
}

func NewEndpointMetrics() *EndpointMetrics {
	return &EndpointMetrics{
		MaxTime:     0,
		MinTime:     math.MaxInt64,
		Count:       0,
		ErrorCount:  0,
		Sent:        0,
		SentSize:    0,
		Receive:     0,
		ReceiveSize: 0,
		StatusCount: sync.NewIntHashMap(),
		TotalTime:   0,
	}
}

func (m *dmsMetrics) GetOrCreateEndpointMetrics(endpoint string) *EndpointMetrics {
	return m.serverResponseTimes.GetOrInsert(endpoint, NewEndpointMetrics()).(*EndpointMetrics)
}

func (m *dmsMetrics) Metrics(interval time.Duration) *metrics.DMSMetrics {
	endpointResponseTimes := make(map[string]metrics.DMSEndpointResponseMetrics, 0)

	totalRequestCount := int64(0)
	totalResponseCount := int64(0)
	totalResponseSize := int64(0)
	totalResponseTime := int64(0)
	totalErrorCount := int64(0)

	for kv := range m.serverResponseTimes.Iter() {
		var endpoint = kv.Key.(string)
		var mt = kv.Value.(*EndpointMetrics)
		var reqCount = atomic.LoadInt64(&(mt.Count))
		if reqCount > 0 {
			var duration int64
			var responseCount int64
			var sentSize int64
			var errorCount = atomic.LoadInt64(&(mt.ErrorCount))

			var endpointResponseData = metrics.DMSEndpointResponseMetrics{
				ErrorCount: errorCount,
				ErrorRate:  float64(errorCount) / float64(reqCount),
			}

			if interval > 0 {
				endpointResponseData.RPM = float64(reqCount) / interval.Seconds() * 60
			}

			if enableAllDMSMetrics {
				duration = atomic.LoadInt64(&(mt.TotalTime))
				responseCount = reqCount - errorCount

				minTime := atomic.LoadInt64(&(mt.MinTime))
				maxTime := atomic.LoadInt64(&(mt.MaxTime))
				endpointResponseData.MinTime = time.Duration(minTime).Seconds()
				endpointResponseData.MaxTime = time.Duration(maxTime).Seconds()
				endpointResponseData.StatusCodeCount = mt.StatusCount.ToMap()

				sentSize = atomic.LoadInt64(&(mt.SentSize))
				if responseCount > 0 {
					endpointResponseData.AverageTime = time.Duration(duration).Seconds() / float64(responseCount)
					endpointResponseData.AverageSize = float64(sentSize) / float64(responseCount)
				}
			}

			endpointResponseTimes[endpoint] = endpointResponseData

			totalRequestCount += reqCount
			totalResponseCount += responseCount
			totalResponseSize += sentSize
			totalResponseTime += duration
			totalErrorCount += errorCount
		}
	}

	requestPerMin := float64(0)
	if interval > 0 {
		requestPerMin = float64(totalRequestCount) / interval.Seconds() * float64(60)
	}

	averageResponseTime := time.Duration(0)
	averageResponseSize := float64(0)
	if totalRequestCount > 0 {
		averageResponseTime = time.Duration(totalResponseTime / totalRequestCount)
		averageResponseSize = float64(totalResponseSize) / float64(totalRequestCount)
	}

	totalErrorRate := float64(0)
	if totalRequestCount > 0 {
		totalErrorRate = float64(totalErrorCount) / float64(totalRequestCount)
	}

	r := &metrics.DMSMetrics{
		RequestPerMin:         requestPerMin,
		AverageResponseSize:   averageResponseSize,
		AverageResponseTime:   averageResponseTime.Seconds(),
		ErrorCount:            totalErrorCount,
		ErrorRate:             totalErrorRate,
		EndpointResponseTimes: endpointResponseTimes,
	}

	r.Label = metrics.DMSMetricsType
	r.ProcStats = reporter.GetProcStats()
	r.Timestamp = time.Now().Unix()
	if enableAllDMSMetrics {
		r.EnableMetrics = "all"
	} else {
		r.EnableMetrics = "error_count,error_rate,rpm"
	}

	return r
}
