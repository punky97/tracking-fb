package bkconsumer

import (
	"github.com/punky97/go-codebase/core/bkconsumer/reporter"
	"github.com/punky97/go-codebase/core/bkconsumer/utils"
	"github.com/punky97/go-codebase/core/logger"
	reporter2 "github.com/punky97/go-codebase/core/reporter"
	"context"
	"fmt"
	"sync"
	"time"

	cron "github.com/robfig/cron/v3"
)

// Scheduler --
type Scheduler struct {
	ScheduleJob []*ScheduleJobOption
	c           *cron.Cron
	ctx         context.Context
	cancelFunc  context.CancelFunc
	lock        sync.Mutex

	schedulerMetrics *reporter.SchedulerWorkerMetrics
}

const (
	// AllowedMaxRetries --
	AllowedMaxRetries int64 = 100
	// DefaultRetries --
	DefaultRetries int64 = 10
	// OneTimeMode one time cron config
	OneTimeMode string = "once"
	// MultipleBoundMode bound by a date
	MultipleBoundMode string = "multiple"
	// EndlessMode --
	EndlessMode string = "endless"
	// DefaultBoundDate default bound date in minute
	DefaultBoundDate = 5
	// ScheduleFail Schedule Job Status
	ScheduleFail int = 1
)

// ScheduleTask --
type ScheduleTask interface {
	// Handle function return value indicates the successfull or failure of the task
	Handle(interface{}) ProcessStatus
	GetName() string
}

type DisableMetricTask interface {
	IsDisableMetric() bool
}

// SchedulerHealthCheck -
//type SchedulerHealthCheck struct {
//	StartTime      int64
//	Cron           string
//	Name           string
//	LastRun        int64
//	LastRunSuccess int64
//}

type ScheduleScrapeInfo struct {
	Name                   string `json:"name"`
	Cron                   string `json:"cron"`
	Type                   string `json:"type"`
	BoundDate              int64  `json:"bound_date"`
	StartTime              int64  `json:"start_time"`
	LastRunningTime        int64  `json:"last_running_time"`
	LastRunningTimeSuccess int64  `json:"last_running_time_success"`
	ScrapeTime             int64  `json:"scrape_time"`
}

// ScheduleJobOption --
type ScheduleJobOption struct {
	cron      string
	st        ScheduleTask
	retries   int64
	timeType  string
	boundDate time.Time
	finished  bool
	id        cron.EntryID
	status    int

	// allow ignore metric for utility tasks
	ignoreMetric bool

	ScheduleScrapeInfo
}

// NewScheduler creates new scheduler
func NewScheduler(appName string, ctx context.Context, cf context.CancelFunc, sj []*ScheduleJobOption) *Scheduler {
	return &Scheduler{
		ScheduleJob: sj,
		c:           cron.New(),
		ctx:         ctx,
		cancelFunc:  cf,
		lock:        sync.Mutex{},

		schedulerMetrics: reporter.NewSchedulerWorkerMetrics(fmt.Sprintf("%v", appName)),
	}
}

func convertCodeToString(code int64) string {
	switch code {
	case utils.ProcessOKCode:
		return "success"
	case utils.ProcessFailDropCode:
		return "drop"
	case utils.ProcessFailRetryCode:
		return "retry"
	case utils.ProcessFailReproduceCode:
		return "fail-reproduce"
	}
	return ""
}

// AddTask --
func (s *Scheduler) AddTask(t *ScheduleJobOption) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.ScheduleJob = append(s.ScheduleJob, t)
}

// Close --
func (s *Scheduler) Close() {
	s.stop()
}

// Stop --
func (s *Scheduler) stop() {
	s.cancelFunc()
	s.schedulerMetrics.Close()
	logger.BkLog.Info("Scheduler closed")
}

// Schedule --
// i can be number of retry, should run if other task is not completed from the last schedule...
func (s *Scheduler) Schedule(runApi func(), i ...interface{}) {
	workerMetrics := s.schedulerMetrics
	workerMetrics.Start()
	for _, v := range s.ScheduleJob {
		vv := v
		st := vv.st
		vv.StartTime = time.Now().Unix()
		vv.Name = st.GetName()
		vv.Cron = vv.cron
		vv.Type = vv.timeType
		vv.BoundDate = vv.boundDate.Unix()

		id, err := s.c.AddFunc(vv.cron, func() {
			vv.LastRunningTime = time.Now().Unix()
			logger.BkLog.Infof("Running job %v at %v", st.GetName(),
				time.Now().Format("2006-01-02T15:04:05.999999-07:00"))
			localMaxTry := vv.retries

			if localMaxTry == 0 {
				localMaxTry = 1
			}

			if localMaxTry < 0 || localMaxTry > AllowedMaxRetries {
				localMaxTry = DefaultRetries
			}
			var retries int64
			var shouldRetry = true
			var rp reporter.SchedulerReporter = reporter.NewEmptySchedulerReporter()
			// scheduler dont have error_rate, error_count
			if !vv.ignoreMetric && reporter2.IsMetricsEnable() && reporter2.IsAllMetricsEnable() {
				rp = reporter.NewSchedulerReporter(st.GetName(), workerMetrics)
			}
			var lastRespCode = utils.ProcessOKCode
			for retries < localMaxTry && shouldRetry {
				retries++
				resp := st.Handle(nil)
				code := resp.Code
				lastRespCode = code
				switch code {
				case utils.ProcessOKCode:
					vv.LastRunningTimeSuccess = vv.LastRunningTime
					logger.BkLog.Infof("Process job %v exit with %v return code at round number %v",
						st.GetName(),
						convertCodeToString(code),
						retries,
					)
					shouldRetry = false
				case utils.ProcessFailDropCode:
					logger.BkLog.Warnf("Process job %v exit with %v return code at round number %v",
						st.GetName(),
						convertCodeToString(code),
						retries,
					)
					shouldRetry = false
				case utils.ProcessFailRetryCode:
					shouldRetry = true
				}
				logger.BkLog.Infof("Process %v exit with code %v at retry number %v",
					st.GetName(),
					convertCodeToString(code),
					retries,
				)
			}
			if retries == localMaxTry && shouldRetry {
				logger.BkLog.Warnf("Process %v exit with error code at retry number %v. Give up",
					st.GetName(),
					retries,
				)
			}

			rp.Handled(int(lastRespCode))
			rp.End()
			logger.BkLog.Infof("Running job %v finished at %v", st.GetName(), time.Now().Format("2006-01-02T15:04:05.999999-07:00"))
			vv.finished = true
		})
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Cannot schedule job %v due to error %v", st.GetName(), err))
			vv.finished = true
			vv.status = ScheduleFail // ready to remove from job list
		} else {
			vv.id = id
		}
	}

	s.c.Start()
	runApi()

	tick := time.Tick(1 * time.Second)

	defer s.c.Stop()
forever:
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			if s.shouldTerminate() {
				break forever
			}
		}
	}
}

// scheduler should be terminated when:
//  + timeType == Once && finished = true
//  + timeType == Multiple && finished = true && current_time >= bound_time
func (s *Scheduler) shouldTerminate() (result bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.ScheduleJob) == 0 {
		result = true
	} else {
		var removedItems []int
		var remaining []*ScheduleJobOption
		for idx, v := range s.ScheduleJob {
			if v.status == ScheduleFail {
				removedItems = append(removedItems, idx)
				continue
			}
			switch v.timeType {
			case OneTimeMode:
				if v.finished {
					removedItems = append(removedItems, idx)
				} else {
					remaining = append(remaining, v)
				}
			case MultipleBoundMode:
				now := time.Now()
				if v.finished && now.After(v.boundDate) {
					removedItems = append(removedItems, idx)
				} else {
					remaining = append(remaining, v)
				}
			default:
				remaining = append(remaining, v)
			}
		}

		for _, idx := range removedItems {
			sj := s.ScheduleJob[idx]
			s.c.Remove(sj.id)
		}
		if len(removedItems) > 0 {
			logger.BkLog.Infof("Removed %v jobs", len(removedItems))
		}
		s.ScheduleJob = remaining
		result = len(s.ScheduleJob) == 0
	}
	return
}
