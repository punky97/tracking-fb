package notifier

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/spf13/cast"
	"time"
)

type SentryNotifier struct {
	dsn string
}

type SentryMessage struct {
	App         string                 `json:"author_name"`
	RefLink     string                 `json:"title_link"`
	NotiDetail  string                 `json:"text"`
	NotiContent string                 `json:"title"`
	LogLevel    sentry.Level           `json:"log_level"`
	Env         string                 `json:"footer"`
	Extra       map[string]interface{} `json:"extra"`
	Tags        map[string]string      `json:"tags"`
}

func NewSentryNotifier(dsn string) *SentryNotifier {
	return &SentryNotifier{
		dsn: dsn,
	}
}

func (n *SentryNotifier) Format(log constant.ReportLog) *Message {
	var logLevel sentry.Level
	if log.Priority == constant.ReportAlert {
		logLevel = sentry.LevelError

	} else if log.Priority == constant.ReportWarn {
		logLevel = sentry.LevelWarning
	} else {
		logLevel = sentry.LevelInfo
	}

	var appName string
	var notiContent string
	var notiDetail string
	var infos = make([]*serviceInfo, 0)
	var env = GetReportEnv()
	switch log.ReportType {
	case constant.DMSTimeoutReport:
		data := GetDMSTimeoutData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("dms timeout: method %v exceeds %v ms", data.Method, data.Timeout)
		notiDetail = fmt.Sprintf("request detail: %v", escape(data.Req))
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.DMSCancelReport:
		data := GetDMSCancelData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("dms cancelled: method %v is cancelled", data.Method)
		notiDetail = fmt.Sprintf("request detail: %v", escape(data.Req))
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.DMSHealthReport:
		data := GetDMSHealthData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("dms health alert: service %v has too many errors", appName)
		for ep, m := range data.Endpoints {
			infos = append(infos, &serviceInfo{
				Info:    ep,
				Value:   fmt.Sprintf("Error rate: %v, Error count: %v, RPM: %v", m.ErrorRate, m.ErrorCount, m.RPM),
				IsShort: false,
			})
		}
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.APITimeoutReport:
		data := GetAPITimeoutData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("api timeout: api %v exceeds %v ms", escape(data.URI), data.Timeout)
		notiDetail = fmt.Sprintf("request: %v %v, from %v", data.Method, escape(data.URI), data.Remote)
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.APIBadGateway:
		data := GetAPIBadGateWayData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("api bad gateway: api %v return 502 (bad gateway)", escape(data.URI))
		notiDetail = fmt.Sprintf("request: %v %v, from %v", data.Method, escape(data.URI), data.Remote)
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.APINoService:
		data := GetAPINoServiceData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("api no service: api %v found no available service", escape(data.Service))
		if len(data.URI) > 0 {
			notiDetail = fmt.Sprintf("request: %v %v, from %v", data.Method, escape(data.URI), data.Remote)
		}
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.APIHealthReport:
		data := GetAPIHealthData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("api health alert: api %v has too many errors", appName)
		notiDetail = fmt.Sprintf("there are %v alert endpoint(s)", len(data.Endpoints))
		for ep, m := range data.Endpoints {
			infos = append(infos, &serviceInfo{
				Info:    ep,
				Value:   fmt.Sprintf("Error rate: %v, Error count: %v, RPM: %v", m.ErrorRate, m.ErrorCount, m.RPM),
				IsShort: false,
			})
		}
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})
	case constant.ConsumerErrorReport:
		data := GetConsumerReportData(log.Data)
		appName = data.Service
		notiContent = fmt.Sprintf("consumer error: failure in handling %v messages", len(data.ErrorMessages))
		for _, m := range data.ErrorMessages {
			infos = append(infos, &serviceInfo{
				Info:    m.Handler,
				Value:   m.Message,
				IsShort: false,
			})
		}
		infos = append(infos, &serviceInfo{
			Info:    "Source",
			Value:   data.Source,
			IsShort: true,
		})

	default:
		if name, ok := log.Data["app"]; ok {
			appName = fmt.Sprintf("%v", name)
		}
		if content, ok := log.Data["desc"]; ok {
			notiContent = fmt.Sprintf("%v", content)
		}
		if detail, ok := log.Data["detail"]; ok {
			notiDetail = fmt.Sprintf("%v", detail)
		}
		if extras, ok := log.Data["extras"].([]map[string]string); ok {
			if len(extras) > 0 {
				for _, extra := range extras {
					infos = append(infos, &serviceInfo{
						Info:    extra["info"],
						Value:   extra["desc"],
						IsShort: false,
					})
				}
			}
		}
		if source, ok := log.Data["source"].(string); ok {
			infos = append(infos, &serviceInfo{
				Info:    "Source",
				Value:   source,
				IsShort: true,
			})
		}
	}

	return &Message{
		Sentry: &SentryMessage{
			App:         appName,
			NotiDetail:  notiDetail,
			LogLevel:    logLevel,
			NotiContent: notiContent,
			Env:         env,
			Extra:       cast.ToStringMap(log.Data["sentry_extra"]),
			Tags:        cast.ToStringMapString(log.Data["sentry_tags"]),
		},
	}
}

func (n *SentryNotifier) MessageData(ms *Message) []byte {
	m := ms.Sentry
	event := sentry.NewEvent()
	event.Message = m.NotiContent
	event.Level = m.LogLevel
	event.Environment = m.Env
	event.Logger = m.NotiDetail
	event.Tags = m.Tags
	event.Extra = m.Extra

	data, err := json.Marshal(event)
	if err == nil {
		return data
	}

	return []byte{}
}

func (n *SentryNotifier) Push(pcm pusherCustomMessage) {
	_ = sentry.Init(sentry.ClientOptions{
		Dsn:   n.dsn,
		Debug: true,
	})

	sentry.CaptureEvent(pcm.sentryMessage.message)
	sentry.Flush(time.Second * 5)
}
