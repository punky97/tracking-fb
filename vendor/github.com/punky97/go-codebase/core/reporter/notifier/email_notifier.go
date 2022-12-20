package notifier

import (
	"bytes"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"fmt"
	"github.com/mailgun/mailgun-go"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"html/template"
	"strings"
)

var templateHtml = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
</head>

<body style="padding: 0; margin: 0; background-color: #FAFAFA; -webkit-font-smoothing: antialiased; font-family: sans-serif; font-size: 13px;">
<p>{{ .notiContent}}</p>
<p>{{ .notiDetail}}</p>
<p>{{ .app}}</p>
<p>{{ .env}}</p>
</body>
</html>`

type EmailNotifier struct {
	mailClient mailgun.Mailgun
	fromMail   string
	toMail     string
}

type EmailMessage struct {
	App         string `json:"author_name"`
	RefLink     string `json:"title_link"`
	NotiDetail  string `json:"text"`
	NotiContent string `json:"title"`
	Env         string `json:"footer"`
	From        string `json:"from"`
	To          string `json:"to"`
	Cc          string `json:"cc"`
	Bcc         string `json:"bcc"`
}

func NewEmailNotifier() *EmailNotifier {
	domain := viper.GetString("notifier.plugins.email.domain")
	apiKey := viper.GetString("notifier.plugins.email.api_key")
	//pubKey := viper.GetString("notifier.plugins.email.public_key")

	return &EmailNotifier{
		fromMail:   viper.GetString("notifier.plugins.email.mail_from"),
		toMail:     viper.GetString("notifier.plugins.email.mail_to"),
		mailClient: mailgun.NewMailgun(domain, apiKey),
	}
}

func (n *EmailNotifier) Format(log constant.ReportLog) *Message {
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

	var mailFrom, mailTo string

	if cast.ToString(log.Data["mail_from"]) == "" {
		mailFrom = n.fromMail
	} else {
		mailFrom = cast.ToString(log.Data["mail_from"])
	}

	if cast.ToString(log.Data["mail_to"]) == "" {
		mailTo = n.toMail
	} else {
		mailTo = cast.ToString(log.Data["mail_to"])
	}

	return &Message{
		Email: &EmailMessage{
			App:         appName,
			NotiDetail:  notiDetail,
			NotiContent: notiContent,
			Env:         env,
			From:        mailFrom,
			To:          mailTo,
			Cc:          ConvertEmailAddressesToString(cast.ToStringSlice(log.Data["mail_cc"])),
			Bcc:         ConvertEmailAddressesToString(cast.ToStringSlice(log.Data["mail_bcc"])),
		},
	}
}

func (n *EmailNotifier) MessageData(ms *Message) []byte {
	m := ms.Email
	pageData := map[string]interface{}{
		"notiContent": m.NotiContent,
		"notiDetail":  m.NotiDetail,
		"app":         m.App,
		"env":         m.Env,
	}
	bodyContent := Generate(templateHtml, pageData)
	return []byte(bodyContent)
}

func (n *EmailNotifier) Push(pcm pusherCustomMessage) {
	message := n.mailClient.NewMessage(pcm.emailMessage.from, pcm.emailMessage.subject, pcm.emailMessage.subject, pcm.emailMessage.to)
	message.SetHtml(pcm.emailMessage.text)
	if pcm.emailMessage.cc != "" {
		message.AddCC(pcm.emailMessage.cc)
	}

	if pcm.emailMessage.bcc != "" {
		message.AddBCC(pcm.emailMessage.bcc)
	}

	r, _, err := n.mailClient.Send(message)
	if err != nil {
		logger.BkLog.Error("cant send mess: ", err)
	}

	logger.BkLog.Info(r)
}

func ConvertEmailAddressesToString(addresses []string) string {
	var addrSlice []string
	for i := 0; i < len(addresses); i++ {
		addr := addresses[i]
		if addr != "" {
			addr = `<` + addr + `>`
		}

		addrSlice = append(addrSlice, addr)
	}

	return strings.Join(addrSlice, ", ")
}

// Generate --
func Generate(templateHTML string, pagedata map[string]interface{}) string {
	tmpl := template.New("page")
	var err error
	if tmpl, err = tmpl.Parse(templateHTML); err != nil {
		fmt.Println(err)
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, pagedata)

	return b.String()
}
