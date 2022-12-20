package metrics

const (
	APIMetricsType       = "api_metrics"
	DMSMetricsType       = "dms_metrics"
	SchedulerMetricsType = "scheduler_metrics"
	ConsumerMetricsType  = "consumer_metrics"
)

// API Metrics
type APIEndpointResponseMetrics struct {
	Name            string          `json:"name,omitempty"`
	MaxTime         float64         `json:"max_time"`
	MinTime         float64         `json:"min_time"`
	AverageTime     float64         `json:"average_time"`
	AverageSize     float64         `json:"average_size"`
	RPM             float64         `json:"rpm"`
	StatusCodeCount map[int64]int64 `json:"resp_code_count"`
	ErrorCount      int64           `json:"error_count"`
	ErrorRate       float64         `json:"error_rate"`
}

// APIMetrics serializable structure
type APIMetrics struct {
	AppMetrics
	UpTime                float64                               `json:"uptime,omitempty"`
	RequestPerMin         float64                               `json:"rpm"`
	AverageResponseTime   float64                               `json:"average_response_time"`
	AverageResponseSize   float64                               `json:"average_response_size"`
	ErrorCount            int64                                 `json:"error_count"`
	ErrorRate             float64                               `json:"error_rate"`
	EndpointResponseTimes map[string]APIEndpointResponseMetrics `json:"endpoint_times"`
}

// DMS Metrics

type DMSEndpointResponseMetrics struct {
	Name            string           `json:"name,omitempty"`
	MaxTime         float64          `json:"max_time,omitempty"`
	MinTime         float64          `json:"min_time,omitempty"`
	AverageTime     float64          `json:"average_time,omitempty"`
	AverageSize     float64          `json:"average_size,omitempty"`
	RPM             float64          `json:"rpm"`
	StatusCodeCount map[string]int64 `json:"resp_code_count,omitempty"`
	ErrorCount      int64            `json:"error_count"`
	ErrorRate       float64          `json:"error_rate"`
}

// DMSMetrics serializable structure
type DMSMetrics struct {
	AppMetrics
	UpTime                float64                               `json:"uptime,omitempty"`
	RequestPerMin         float64                               `json:"rpm"`
	AverageResponseTime   float64                               `json:"average_response_time,omitempty"`
	AverageResponseSize   float64                               `json:"average_response_size,omitempty"`
	ErrorCount            int64                                 `json:"error_count"`
	ErrorRate             float64                               `json:"error_rate"`
	EndpointResponseTimes map[string]DMSEndpointResponseMetrics `json:"endpoint_times,omitempty"`
}

// Scheduler metrics
type SchedulerHandlerMetrics struct {
	Name           string  `json:"name,omitempty"`
	AverageRuntime float64 `json:"average_runtime,omitempty"`
	ExecuteCount   int64   `json:"execute_count,omitempty"`
	FailRate       float64 `json:"fail_rate,omitempty"`
	FailCounter    int64   `json:"fail_counter,omitempty"`
}

type SchedulerMetrics struct {
	AppMetrics
	AverageRuntime float64                            `json:"average_process_runtime,omitempty"`
	ExecuteCount   int64                              `json:"total_execute,omitempty"`
	OutputCount    int64                              `json:"total_output,omitempty"`
	FailRate       float64                            `json:"fail_rate,omitempty"`
	FailCounter    int64                              `json:"fail_counter,omitempty"`
	HandlerMetrics map[string]SchedulerHandlerMetrics `json:"handlers,omitempty"`
}

// Consumer metrics
type ConsumerTaskMetrics struct {
	Name               string          `json:"name,omitempty"`
	AverageProcessTime float64         `json:"average_time,omitempty"`
	MinTime            float64         `json:"min_time,omitempty"`
	MaxTime            float64         `json:"max_time,omitempty"`
	StatusCounter      map[int64]int64 `json:"status_count,omitempty"`
	ConsumerRate       float64         `json:"consume_rate"`
	ErrorRate          float64         `json:"error_rate"`
	ErrorCounter       int64           `json:"error_count"`
}

type ConsumerMetrics struct {
	AppMetrics
	AverageProcessTime float64                        `json:"average_process_time,omitempty"`
	ConsumeRate        float64                        `json:"consume_rate"`
	ErrorRate          float64                        `json:"error_rate"`
	ErrorCounter       int64                          `json:"error_count"`
	Tasks              map[string]ConsumerTaskMetrics `json:"tasks"`
}

// App metrics
type ProcStats struct {
	MemUsage     float64 `json:"mem"`
	CPUUsage     float64 `json:"cpu"`
	NumGoRoutine int     `json:"num_goroutine"`
}

type AppMetrics struct {
	Label         string    `json:"label"`
	App           string    `json:"app"`
	PodName       string    `json:"pod_name"`
	ProcStats     ProcStats `json:"proc_stats,omitempty"`
	Timestamp     int64     `json:"ts,omitempty"`
	EnableMetrics string    `json:"enable_metrics,omitempty"`
}
