package middleware

import (
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils/sync"
	"encoding/json"
	"github.com/spf13/viper"
	"math"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	enableAllAPIMetrics = false
)

// NegroniBkStatsMid
type NegroniBkStatsMid struct {
	startReport chan struct{}
	closed      chan struct{}
	Uptime      time.Time
	Pid         int
	appName     string
	podName     string

	reportMetrics  *apiMetrics
	reportInterval time.Duration

	alertMetrics  *apiMetrics
	alertInterval time.Duration
}

type apiMetrics struct {
	EndpointData *sync.HashMap
}

type EndpointMetrics struct {
	MaxTime     int64
	MinTime     int64
	Count       int64
	ErrorCount  int64
	Size        int64
	StatusCount *sync.IntIntMap
	TotalTime   int64
}

func NewNegroniBkStatsMid(appName string) *NegroniBkStatsMid {
	intervalTime := viper.GetInt64("reporter.interval")
	if intervalTime <= 0 {
		intervalTime = constant.DefaultReportTime
	}

	alertIntervalTime := viper.GetInt64("reporter.alert_interval")
	if alertIntervalTime <= 0 {
		alertIntervalTime = constant.DefaultAlertTime
	}

	appPodName := config.GetHostName()
	stats := &NegroniBkStatsMid{
		startReport:    make(chan struct{}, 1),
		closed:         make(chan struct{}, 1),
		Uptime:         time.Now(),
		Pid:            os.Getpid(),
		appName:        appName,
		podName:        appPodName,
		reportInterval: time.Duration(intervalTime) * time.Millisecond,
		alertInterval:  time.Duration(alertIntervalTime) * time.Millisecond,
	}

	stats.reportMetrics = &apiMetrics{
		EndpointData: sync.NewHashMap(),
	}

	stats.alertMetrics = &apiMetrics{
		EndpointData: sync.NewHashMap(),
	}

	if !reporter.IsMetricsEnable() {
		return stats
	}

	enableAllAPIMetrics = reporter.IsAllMetricsEnable()

	alertConfigs := reporter.GetAPIDMSAlertConfig(appName)
	slackNoti := notifier.DefaultNotifier()
	reporter.GetProducerFromConf()

	// report metrics routine
	go func() {
		select {
		case <-stats.closed:
			return
		case <-stats.startReport:
			break
		}

		ticker := time.NewTicker(stats.reportInterval)
		defer ticker.Stop()

		var lastTimeReport = time.Now().Unix()
		for {
			select {
			case <-stats.closed:
				data := stats.GetReportMetrics(time.Since(time.Unix(lastTimeReport, 0)))
				if data.RequestPerMin > 0 {
					dataBytes, _ := json.Marshal(data)
					logger.BkLog.Infof("Metrics: %v", string(dataBytes))
					reporter.PublishSimple("core_app_metrics", dataBytes)
				}

				return
			case <-ticker.C:
				data := stats.GetReportMetrics(stats.reportInterval)
				dataBytes, _ := json.Marshal(data)
				logger.BkLog.Infof("Metrics: %v", string(dataBytes))
				reporter.PublishSimple("core_app_metrics", dataBytes)

				stats.reportMetrics.ResetIntervalCounts()
				lastTimeReport = time.Now().Unix()
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// alert metrics routine
	go func() {
		select {
		case <-stats.closed:
			return
		case <-stats.startReport:
			break
		}

		ticker := time.NewTicker(stats.alertInterval)
		defer ticker.Stop()

		var lastTimeAlert = time.Now().Unix()
		for {
			select {
			case <-stats.closed:
				data := stats.GetAlertMetrics(time.Since(time.Unix(lastTimeAlert, 0)))
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
						ReportType: constant.APIHealthReport,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"service":   appName,
							"source":    appPodName,
							"endpoints": endpoints,
						},
					}))
					dataBytes, _ := json.Marshal(endpoints)
					logger.BkLog.Infof("Alert API %v: %v", appName, string(dataBytes))
				}

				return
			case <-ticker.C:
				data := stats.GetAlertMetrics(stats.alertInterval)
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
						ReportType: constant.APIHealthReport,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"service":   appName,
							"source":    appPodName,
							"endpoints": endpoints,
						},
					}))
					dataBytes, _ := json.Marshal(endpoints)
					logger.BkLog.Infof("Alert API %v: %v", appName, string(dataBytes))
				}

				stats.alertMetrics.ResetIntervalCounts()
				lastTimeAlert = time.Now().Unix()
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return stats
}

func (mw *NegroniBkStatsMid) Close() {
	close(mw.closed)
}

func (mw *NegroniBkStatsMid) Start() {
	close(mw.startReport)
}

// ResetIntervalCounts reset the interval counts
func (m *apiMetrics) ResetIntervalCounts() {
	var keys = make([]string, 0)
	for k := range m.EndpointData.Iter() {
		keys = append(keys, k.Key.(string))
	}

	for _, k := range keys {
		m.EndpointData.Set(k, NewEndpointMetrics())
	}
}

func NewEndpointMetrics() *EndpointMetrics {
	return &EndpointMetrics{
		MinTime:     math.MaxInt64,
		MaxTime:     0,
		Count:       0,
		ErrorCount:  0,
		StatusCount: sync.NewIntIntMap(),
		TotalTime:   0,
	}
}

// Negroni compatible interface
func (mw *NegroniBkStatsMid) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	beginning, recorder := mw.Begin(w)

	next(recorder, r)

	mw.End(beginning, recorder)
}

// Begin starts a recorder
func (mw *NegroniBkStatsMid) Begin(w http.ResponseWriter) (time.Time, transhttp.ResponseWriter) {
	start := time.Now()

	writer := transhttp.NewRecorderResponseWriter(w, 200)

	return start, writer
}

// UpdateMetrics closes the recorder with a specific status
func (mw *apiMetrics) UpdateMetrics(status int, start time.Time, size int, endpoint string) {
	end := time.Now()

	responseTime := end.Sub(start)

	if len(endpoint) > 0 {
		var endpointEntry *EndpointMetrics
		endpointEntry = mw.EndpointData.GetOrInsert(endpoint, NewEndpointMetrics()).(*EndpointMetrics)

		atomic.AddInt64(&(endpointEntry.Count), 1)

		if enableAllAPIMetrics {
			endpointEntry.StatusCount.Increase(int64(status), 1)
		}

		// do not count on error in response time and response size
		if IsErrorCode(status) {
			atomic.AddInt64(&(endpointEntry.ErrorCount), 1)
		} else {
			if enableAllAPIMetrics {
				nanoTime := responseTime.Nanoseconds()
				maxTime := atomic.LoadInt64(&(endpointEntry.MaxTime))
				minTime := atomic.LoadInt64(&(endpointEntry.MinTime))
				if nanoTime > maxTime {
					atomic.StoreInt64(&(endpointEntry.MaxTime), nanoTime)
				}

				if nanoTime < minTime {
					atomic.StoreInt64(&(endpointEntry.MinTime), nanoTime)
				}

				atomic.AddInt64(&(endpointEntry.TotalTime), nanoTime)
				atomic.AddInt64(&(endpointEntry.Size), int64(size))
			}
		}
	}
}

// End closes the recorder with the recorder status
func (mw *NegroniBkStatsMid) End(start time.Time, recorder transhttp.ResponseWriter) {
	endpoint := recorder.Header().Get(transhttp.RoutePatternHeader)

	mw.reportMetrics.UpdateMetrics(recorder.Status(), start, recorder.Size(), endpoint)
	mw.alertMetrics.UpdateMetrics(recorder.Status(), start, recorder.Size(), endpoint)
}

func (mw *NegroniBkStatsMid) GetReportMetrics(interval time.Duration) *metrics.APIMetrics {
	mt := mw.reportMetrics.Metrics(interval)
	mt.App = mw.appName
	mt.PodName = mw.podName

	return mt
}

func (mw *NegroniBkStatsMid) GetAlertMetrics(interval time.Duration) *metrics.APIMetrics {
	mt := mw.alertMetrics.Metrics(interval)
	mt.App = mw.appName
	mt.PodName = mw.podName

	return mt
}

// APIMetrics returns the data serializable structure
func (m *apiMetrics) Metrics(interval time.Duration) *metrics.APIMetrics {
	endpointResponseTimes := make(map[string]metrics.APIEndpointResponseMetrics)

	totalRequestCount := int64(0)
	totalResponseCount := int64(0)
	totalResponseSize := int64(0)
	totalResponseTime := int64(0)
	totalErrorCount := int64(0)

	var endpoint string
	var entry *EndpointMetrics
	for kv := range m.EndpointData.Iter() {
		endpoint = kv.Key.(string)
		entry = kv.Value.(*EndpointMetrics)
		reqCount := atomic.LoadInt64(&(entry.Count))
		if reqCount > 0 {
			var responseCount int64
			var duration int64
			var size int64
			var errorCount = atomic.LoadInt64(&(entry.ErrorCount))

			var endpointResponseData = metrics.APIEndpointResponseMetrics{
				ErrorCount: errorCount,
				ErrorRate:  float64(errorCount) / float64(reqCount),
			}

			if interval > 0 {
				endpointResponseData.RPM = float64(reqCount) / interval.Seconds() * 60
			}

			if enableAllAPIMetrics {
				duration = atomic.LoadInt64(&(entry.TotalTime))
				responseCount = reqCount - errorCount

				minTime := atomic.LoadInt64(&(entry.MinTime))
				maxTime := atomic.LoadInt64(&(entry.MaxTime))
				endpointResponseData.MinTime = time.Duration(minTime).Seconds()
				endpointResponseData.MaxTime = time.Duration(maxTime).Seconds()
				endpointResponseData.StatusCodeCount = entry.StatusCount.ToMap()

				size = atomic.LoadInt64(&(entry.Size))
				if responseCount > 0 {
					endpointResponseData.AverageTime = time.Duration(duration / responseCount).Seconds()
					endpointResponseData.AverageSize = float64(size) / float64(responseCount)
				}
			}

			endpointResponseTimes[endpoint] = endpointResponseData

			totalRequestCount += reqCount
			totalResponseCount += responseCount
			totalResponseSize += size
			totalResponseTime += duration
			totalErrorCount += errorCount
		}
	}

	requestPerMin := float64(0)
	if interval > 0 {
		requestPerMin = float64(totalRequestCount) / interval.Seconds() * 60
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

	r := &metrics.APIMetrics{
		RequestPerMin:         requestPerMin,
		AverageResponseTime:   averageResponseTime.Seconds(),
		AverageResponseSize:   averageResponseSize,
		ErrorCount:            totalErrorCount,
		ErrorRate:             totalErrorRate,
		EndpointResponseTimes: endpointResponseTimes,
	}

	r.Label = metrics.APIMetricsType
	r.ProcStats = reporter.GetProcStats()
	r.Timestamp = time.Now().Unix()
	if enableAllAPIMetrics {
		r.EnableMetrics = "all"
	} else {
		r.EnableMetrics = "error_count,error_rate,rpm"
	}

	return r
}
func IsErrorCode(code int) bool {
	if code >= 200 && code < 400 {
		return false
	}

	return true
}
