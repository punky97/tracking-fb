package reporter

import (
	"time"
)

type SchedulerReporter interface {
	End()
	Handled(int)
}

type schedulerReporter struct {
	handlerName string
	startTime   time.Time
	metrics     *SchedulerWorkerMetrics
	handlerMetrics *handlerMetrics
}

func NewSchedulerReporter(handlerName string, m *SchedulerWorkerMetrics) *schedulerReporter {
	reporter := &schedulerReporter{
		handlerName: handlerName,
		startTime:   time.Now(),
		metrics:     m,
	}

	handlerMetrics := m.metrics.GetOrCreateHandlerMetrics(handlerName)
	reporter.handlerMetrics = handlerMetrics
	return reporter
}

func (r *schedulerReporter) End() {
	r.handlerMetrics.End(time.Since(r.startTime))
}

func (r *schedulerReporter) Handled(code int) {
	r.handlerMetrics.Handled(code)
}

type emptySchedulerReporter struct{}

func NewEmptySchedulerReporter() *emptySchedulerReporter {
	return &emptySchedulerReporter{}
}

func (r *emptySchedulerReporter) End() {}
func (r *emptySchedulerReporter) Handled(code int) {}