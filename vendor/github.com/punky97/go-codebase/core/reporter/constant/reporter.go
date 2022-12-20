package constant

type AlertConfig struct {
	Name        string
	ErrorRate   float64
	RequestFrom int64
	RequestTo   int64
}

type ReportLog struct {
	ReportType string                 `json:"report_type,omitempty"`
	Priority   string                 `json:"priority,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

const (
	DefaultReportTime = 2 * 60 * 1000  //ms
	DefaultAlertTime  = 10 * 60 * 1000 //ms

	// report priority
	ReportAlert   = "alert"
	ReportResolve = "resolve"
	ReportWarn    = "warn"
	ReportInfo    = "info"

	PluginSentry = "sentry"
	PluginSlack  = "slack"
	PluginEmail  = "email"

	// report type
	DMSTimeoutReport = "dms_timeout"
	DMSCancelReport  = "dms_cancel"
	DMSHealthReport  = "dms_health_alert"

	APITimeoutReport = "api_timeout"
	APIBadGateway    = "api_bad_gateway"
	APINoService     = "api_no_service"
	APIHealthReport  = "api_health_alert"

	ConsumerErrorReport = "consumer_error"
	CashierErrorReport  = "cashier_error_report"

	PaymentUpdateFail = "payment_update_fail"

	CriticalError     = "critical_error"
	ServiceInPressure = "service_in_pressure"
)
