package notifier

import (
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/metrics"
	"encoding/json"
	"github.com/spf13/viper"
)

var (
	defaultNotifier  *SlackNotifier
	criticalNotifier *SlackNotifier
	closeChan        chan bool = make(chan bool)
)

func DefaultNotifier() *SlackNotifier {
	if defaultNotifier == nil {
		defaultNotifier = NewSlackNotifier()
	}

	return defaultNotifier
}

func CriticalNotifier() *SlackNotifier {
	if criticalNotifier == nil {
		criticalNotifier = NewSlackCriticalNotifier()
	}

	return criticalNotifier
}

func Close() {
	close(closeChan)
}

type Message struct {
	Hash   string
	Slack  *SlackMessage
	Sentry *SentryMessage
	Email  *EmailMessage
}

type Notifier interface {
	Format(log constant.ReportLog) *Message
	MessageData(m *Message) []byte
	Push(message pusherCustomMessage)
}

type DMSReportData struct {
	Service string
	Method  string
	Timeout float64
	Req     string
	Source  string
}

type DMSHealthData struct {
	Service   string
	Source    string
	Endpoints map[string]metrics.DMSEndpointResponseMetrics
}

type APIReportData struct {
	Service string
	Method  string
	Timeout float64
	Remote  string
	URI     string
	Source  string
}

type APIHealthData struct {
	Service   string
	Source    string
	Endpoints map[string]metrics.APIEndpointResponseMetrics
}

type ConsumerErrorMessage struct {
	Message string `json:"message"`
	Handler string `json:"task"`
}
type ConsumerReportData struct {
	Service       string
	Source        string
	ErrorMessages []ConsumerErrorMessage
}

func GetReportEnv() string {
	return viper.GetString("reporter.env")
}

func GetDMSTimeoutData(data map[string]interface{}) DMSReportData {
	serviceName, _ := data["service"].(string)
	methodName, _ := data["method"].(string)
	timeoutNs, _ := data["timeout"].(int64)
	req, _ := json.Marshal(data["req"])
	source, _ := data["source"].(string)
	return DMSReportData{
		Service: serviceName,
		Method:  methodName,
		Timeout: float64(timeoutNs) / 1e6,
		Req:     string(req),
		Source:  string(source),
	}
}

func GetDMSCancelData(data map[string]interface{}) DMSReportData {
	serviceName, _ := data["service"].(string)
	methodName, _ := data["method"].(string)
	req, _ := json.Marshal(data["req"])
	source, _ := data["source"].(string)
	return DMSReportData{
		Service: serviceName,
		Source:  string(source),
		Method:  methodName,
		Req:     string(req),
	}
}

func GetDMSHealthData(data map[string]interface{}) DMSHealthData {
	serviceName, _ := data["service"].(string)
	endpoints, _ := data["endpoints"].(map[string]interface{})
	source, _ := data["source"].(string)
	endpointData := make(map[string]metrics.DMSEndpointResponseMetrics)

	for k, v := range endpoints {
		mt := v.(metrics.DMSEndpointResponseMetrics)
		endpointData[k] = mt
	}

	return DMSHealthData{
		Service:   serviceName,
		Source:    source,
		Endpoints: endpointData,
	}
}

func GetAPITimeoutData(data map[string]interface{}) APIReportData {
	serviceName, _ := data["service"].(string)
	methodName, _ := data["method"].(string)
	source, _ := data["source"].(string)
	uri, _ := data["uri"].(string)
	remote, _ := data["remote"].(string)
	timeoutNs, _ := data["timeout"].(int64)
	return APIReportData{
		Service: serviceName,
		Method:  methodName,
		Source:  source,
		Remote:  remote,
		URI:     uri,
		Timeout: float64(timeoutNs) / 1e6,
	}
}

func GetAPIBadGateWayData(data map[string]interface{}) APIReportData {
	serviceName, _ := data["service"].(string)
	methodName, _ := data["method"].(string)
	uri, _ := data["uri"].(string)
	remote, _ := data["remote"].(string)
	source, _ := data["source"].(string)
	return APIReportData{
		Service: serviceName,
		Method:  methodName,
		Remote:  remote,
		URI:     uri,
		Source:  source,
	}
}

func GetAPIHealthData(data map[string]interface{}) APIHealthData {
	serviceName, _ := data["service"].(string)
	source, _ := data["source"].(string)
	endpoints, _ := data["endpoints"].(map[string]interface{})
	endpointData := make(map[string]metrics.APIEndpointResponseMetrics)

	for k, v := range endpoints {
		mt := v.(metrics.APIEndpointResponseMetrics)
		endpointData[k] = mt
	}

	return APIHealthData{
		Service:   serviceName,
		Source:    source,
		Endpoints: endpointData,
	}
}

func GetAPINoServiceData(data map[string]interface{}) APIReportData {
	serviceName, _ := data["service"].(string)
	methodName, _ := data["method"].(string)
	uri, _ := data["uri"].(string)
	remote, _ := data["remote"].(string)
	source, _ := data["source"].(string)
	return APIReportData{
		Service: serviceName,
		Source:  source,
		Method:  methodName,
		Remote:  remote,
		URI:     uri,
	}
}

func GetConsumerReportData(data map[string]interface{}) ConsumerReportData {
	serviceName, _ := data["service"].(string)
	source, _ := data["source"].(string)
	messages, _ := data["messages"].([]interface{})
	errorMessages := make([]ConsumerErrorMessage, 0)
	for _, m := range messages {
		dataBytes, _ := json.Marshal(m)
		var c ConsumerErrorMessage
		_ = json.Unmarshal(dataBytes, &c)
		errorMessages = append(errorMessages, c)
	}

	return ConsumerReportData{
		Service:       serviceName,
		Source:        source,
		ErrorMessages: errorMessages,
	}
}
