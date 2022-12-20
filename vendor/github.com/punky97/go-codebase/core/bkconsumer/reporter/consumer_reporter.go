package reporter

import (
	"time"
)

type ConsumerReporter interface {
	End(time.Time)
	Handled(int)
	Report([]byte)
}

type consumerReporter struct {
	taskName  string
	startTime time.Time
	metrics   *ConsumerWorkerMetrics
	taskMetrics *taskMetrics
}

func NewConsumerReporter(handlerName string, m *ConsumerWorkerMetrics) *consumerReporter {
	reporter := &consumerReporter{
		taskName:  handlerName,
		startTime: time.Now(),
		metrics:   m,
	}

	reporter.taskMetrics = m.metrics.GetOrCreateTaskMetrics(handlerName)

	return reporter
}

func (r *consumerReporter) End(start time.Time) {
	r.taskMetrics.End(time.Since(start))
}

func (r *consumerReporter) Handled(code int) {
	r.taskMetrics.Handled(code)
}

func (r *consumerReporter) Report(msg []byte) {
	r.taskMetrics.Report(r.taskName, msg)
}

type emptyConsumerReporter struct{}

func NewEmptyConsumerReporter() *emptyConsumerReporter {
	return &emptyConsumerReporter{}
}

func (c *emptyConsumerReporter) End(t time.Time) {

}

func (c *emptyConsumerReporter) Handled(code int) {

}

func (c *emptyConsumerReporter) Report(s []byte) {

}