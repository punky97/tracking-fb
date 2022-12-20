package notifier

import (
	"bytes"
	"github.com/punky97/go-codebase/core/cache/local"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/utils"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"math/rand"
)

const _maxLocalCache int = 1000

var (
	netClient = &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	lock          sync.RWMutex
	notifyPusher  *slackNotifierPusher
	membersPicked = make(map[string]string)
)

type serviceInfo struct {
	Info    string `json:"title"`
	Value   string `json:"value"`
	IsShort bool   `json:"short"`
}

type SlackMessage struct {
	TaggingMsg  string         `json:"pretext"`
	Color       string         `json:"color"`
	App         string         `json:"author_name"`
	AppLink     string         `json:"author_link"`
	NotiContent string         `json:"title"`
	RefLink     string         `json:"title_link"`
	NotiDetail  string         `json:"text"`
	Infos       []*serviceInfo `json:"fields"`
	Env         string         `json:"footer"`
	Ts          int64          `json:"ts"`
	members     []string       `json:"-"`
}

type SlackNotifier struct {
	webhook    string
	notifyChan chan []byte

	lazyNoti   bool
	localCache *local.BkLocalCache

	members []string
}

type pusherMessage struct {
	webhook string
	message []byte
}

type slackNotifierPusher struct {
	pushChan chan pusherMessage
}

func init() {
	lock.Lock()
	defer lock.Unlock()

	if notifyPusher == nil {
		notifyPusher = NewSlackNotifyPusher()
	}
}

func NewSlackNotifier() *SlackNotifier {
	return NewSlackNotifierCustom(viper.GetString("reporter.notifier.slack.webhook"))
}

func NewSlackCriticalNotifier() *SlackNotifier {
	return NewSlackNotifierCustom(viper.GetString("reporter.notifier.critical.slack.webhook"))
}

func NewSlackNotificationCustomWithMembers(customWebhook string, members []string) *SlackNotifier {
	slackNotifier := &SlackNotifier{
		webhook:    customWebhook,
		lazyNoti:   true,
		localCache: local.NewLocalCache(_maxLocalCache),
		members:    members,
	}

	return slackNotifier
}

func NewSlackNotifierCustom(customWebhook string) *SlackNotifier {
	slackNotifier := &SlackNotifier{
		webhook:    customWebhook,
		lazyNoti:   true,
		localCache: local.NewLocalCache(_maxLocalCache),
	}

	return slackNotifier
}

func (n *SlackNotifier) Format(log constant.ReportLog) *Message {
	var color string
	if log.Priority == constant.ReportAlert {
		color = "#ff0000"
	} else {
		color = "#ffa64f"
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
		Slack: &SlackMessage{
			Color:       color,
			App:         appName,
			NotiContent: notiContent,
			NotiDetail:  notiDetail,
			Infos:       infos,
			Env:         env,
			Ts:          time.Now().Unix(),
			members:     n.members,
		},
	}
}

func (n *SlackNotifier) Hash(m *SlackMessage) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", m.Env, m.App, m.NotiContent, utils.GetMD5Hash(m.NotiDetail), m.RefLink)
}

func (n *SlackNotifier) MessageData(ms *Message) []byte {
	m := ms.Slack
	mm := map[string]interface{}{
		"attachments": []interface{}{m},
	}

	// Assign randomly to member
	totalMembers := len(m.members)

	if totalMembers > 0 {
		// Reset picked members
		if len(membersPicked) == totalMembers {
			membersPicked = make(map[string]string)
		}

		// Waiting for the picking rounds complete before picking a picked member again
		for i := 0; i < totalMembers; i++ {
			member := m.members[rand.Intn(totalMembers)]
			if _, exist := membersPicked[member]; !exist {
				membersPicked[member] = member
				mm["text"] = fmt.Sprintf("Hey <%s>, Please fix the error bellow and reply to the thread.", member)
				break
			}
		}
	}

	data, err := json.Marshal(mm)
	if err == nil {
		return data
	}

	return []byte{}
}

func (n *SlackNotifier) Notify(m *Message) {
	if len(n.webhook) == 0 {
		return
	}

	if n.lazyNoti {
		_, h, err := n.localCache.Get(n.Hash(m.Slack))
		if err != nil {
			logger.BkLog.Errorw("Error when get local cache", "mHash()", n.Hash(m.Slack), "err", err)
		} else {
			if h == local.CacheCodeHit {
				logger.BkLog.Warnf("DO NOT notify message with same description: %v", string(n.MessageData(m)))
				return
			}
		}
	}

	if notifyPusher != nil {
		select {
		case notifyPusher.pushChan <- pusherMessage{webhook: n.webhook, message: n.MessageData(m)}:
			if n.lazyNoti {
				_ = n.localCache.Set(n.Hash(m.Slack), 1, time.Second*60)
			}
			break

			// publish message with timeout
		case <-time.After(time.Duration(10) * time.Second):
			break
		}
	}
}

func escape(mess string) string {
	mess = strings.Replace(mess, "&", "&amp;", -1)
	mess = strings.Replace(mess, "<", "&lt;", -1)
	mess = strings.Replace(mess, ">", "&gt;", -1)

	return mess
}

func NewSlackNotifyPusher() *slackNotifierPusher {
	pusher := &slackNotifierPusher{
		pushChan: make(chan pusherMessage, 1000),
	}

	go func() {
		for {
			select {
			case <-closeChan:
				return
			case msg := <-pusher.pushChan:
				buf := bytes.NewBuffer(msg.message)
				_, err := netClient.Post(msg.webhook, "application/json", buf)
				if err != nil {
					logger.BkLog.Errorw("Error when notify on slack", "err", err)
				}
			}
		}
	}()

	return pusher
}

func (n *SlackNotifier) Push(pcm pusherCustomMessage) {
	msg := pcm.slackMessage
	buf := bytes.NewBuffer(msg.message)
	_, err := netClient.Post(msg.webhook, "application/json", buf)
	if err != nil {
		logger.BkLog.Errorw("Error when notify on slack", "err", err)
	}
}
