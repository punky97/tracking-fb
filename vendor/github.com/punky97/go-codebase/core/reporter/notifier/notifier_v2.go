package notifier

import (
	"github.com/punky97/go-codebase/core/cache/local"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/utils"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"sync"
	"time"
)

const _maxLocalCacheV2 int = 1000
const _pressureThreshold = 800

var (
	lockV2         sync.RWMutex
	notifyV2Pusher *notifierPusher
)

func init() {
	lockV2.Lock()
	defer lockV2.Unlock()

	if notifyV2Pusher == nil {
		notifyV2Pusher = NewNotifyV2Pusher()
	}
}

type pusherCustomMessage struct {
	hash          string
	slackMessage  slackMessage
	sentryMessage sentryMessage
	emailMessage  emailMessage
}

type notifierPusher struct {
	pushChan chan pusherCustomMessage
}

type slackMessage struct {
	notifier Notifier
	enable   bool
	webhook  string
	message  []byte
}

type sentryMessage struct {
	notifier Notifier
	enable   bool
	dsn      string
	message  *sentry.Event
}

type emailMessage struct {
	notifier Notifier
	enable   bool
	from     string
	to       string
	cc       string
	bcc      string
	subject  string
	text     string
}

type Noti struct {
	Notifier Notifier
	Enable   bool
	Path     string
}

type NotiV2 struct {
	Slack      Noti
	Sentry     Noti
	Email      Noti
	lazyNoti   bool
	localCache *local.BkLocalCache
}

func NewNotifyV2() *NotiV2 {
	notiV2 := &NotiV2{
		localCache: local.NewLocalCache(_maxLocalCacheV2),
	}

	pluginConf := viper.GetStringMap("notifier.plugins")

	for pluginName := range pluginConf {
		if !viper.GetBool(fmt.Sprintf("notifier.plugins.%v.enable", pluginName)) {
			continue
		}

		switch pluginName {
		case constant.PluginSlack:
			notiV2.Slack.Enable = true

			notiV2.Slack.Path = viper.GetString("notifier.plugins.slack.webhook")
			notiV2.Slack.Notifier = NewSlackNotifierCustom(notiV2.Slack.Path)

		case constant.PluginSentry:
			notiV2.Sentry.Enable = true
			notiV2.Sentry.Path = viper.GetString("notifier.plugins.sentry.dsn")
			notiV2.Sentry.Notifier = NewSentryNotifier(notiV2.Sentry.Path)

		case constant.PluginEmail:
			notiV2.Email.Enable = true
			notiV2.Email.Notifier = NewEmailNotifier()

		}
	}

	return notiV2
}

func (n *NotiV2) Format(rl constant.ReportLog) *Message {
	m := &Message{}

	if n.Slack.Enable {
		m.Slack = n.Slack.Notifier.Format(rl).Slack
		m.Hash = n.Hash(m.Slack.Env, m.Slack.App, m.Slack.NotiContent, m.Slack.NotiDetail, m.Slack.RefLink)

	}

	if n.Sentry.Enable {
		m.Sentry = n.Sentry.Notifier.Format(rl).Sentry
		m.Hash = n.Hash(m.Sentry.Env, m.Sentry.App, m.Sentry.NotiContent, m.Sentry.NotiDetail, m.Sentry.RefLink)
	}

	if n.Email.Enable {
		m.Email = n.Email.Notifier.Format(rl).Email
		m.Hash = n.Hash(m.Email.Env, m.Email.App, m.Email.NotiContent, m.Email.NotiDetail, m.Email.RefLink)
	}

	return m
}

func (n *NotiV2) Hash(env, app, notifyContent, notifyDetail, refLink string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", env, app, notifyContent, utils.GetMD5Hash(notifyDetail), refLink)
}

func (n *NotiV2) Push(msg pusherCustomMessage) {
	if n.Slack.Enable {
		msg.slackMessage.notifier.Push(msg)
	}

	if n.Sentry.Enable {
		msg.sentryMessage.notifier.Push(msg)
	}
}

func (n *NotiV2) NotifyCustom(m *Message) {
	if m.Hash == "" {
		return
	}

	pcm := pusherCustomMessage{}

	if n.Slack.Enable {
		if len(n.Slack.Path) == 0 {
			return
		}

		pcm.slackMessage.message = n.Slack.Notifier.MessageData(m)
		pcm.slackMessage.enable = true
		pcm.slackMessage.notifier = n.Slack.Notifier
		pcm.slackMessage.webhook = n.Slack.Path
	}

	if n.Sentry.Enable {
		if len(n.Sentry.Path) == 0 {
			return
		}

		se := &sentry.Event{}
		_ = json.Unmarshal(n.Sentry.Notifier.MessageData(m), se)
		pcm.sentryMessage.message = se
		pcm.sentryMessage.enable = true
		pcm.sentryMessage.notifier = n.Sentry.Notifier
		pcm.sentryMessage.dsn = n.Sentry.Path
	}

	if n.Email.Enable {
		if len(m.Email.From) == 0 || len(m.Email.To) == 0 {
			return
		}

		pcm.emailMessage.from = m.Email.From
		pcm.emailMessage.to = m.Email.To
		pcm.emailMessage.cc = m.Email.Cc
		pcm.emailMessage.bcc = m.Email.Bcc
		pcm.emailMessage.enable = true
		pcm.emailMessage.notifier = n.Email.Notifier
		pcm.emailMessage.text = string(n.Email.Notifier.MessageData(m))
		pcm.emailMessage.subject = m.Email.NotiContent
	}

	if n.lazyNoti {
		_, h, err := n.localCache.Get(m.Hash)
		if err != nil {
			logger.BkLog.Errorw("Error when get local cache", "mHash()", m.Hash, "err", err)
		} else {
			if h == local.CacheCodeHit {
				logger.BkLog.Warnf("DO NOT notify message with same description")
				return
			}
		}
	}

	if notifyV2Pusher != nil {
		if len(notifyPusher.pushChan) >= _pressureThreshold {
			logger.BkLog.Warnf("Notifier is in pressure, will ignore message")
			return
		}

		select {
		case notifyV2Pusher.pushChan <- pcm:
			if n.lazyNoti {
				_ = n.localCache.Set(m.Hash, 1, time.Second*60)
			}
			break

			// publish message with timeout
		case <-time.After(time.Duration(1) * time.Second):
			break
		}
	}
}

func NewNotifyV2Pusher() *notifierPusher {
	pusherCustom := &notifierPusher{
		pushChan: make(chan pusherCustomMessage, 1000),
	}

	go func() {
		for {
			select {
			case <-closeChan:
				logger.BkLog.Info("close")
				return
			case msg := <-pusherCustom.pushChan:
				if msg.slackMessage.enable {
					msg.slackMessage.notifier.Push(msg)
				}

				if msg.sentryMessage.enable {
					msg.sentryMessage.notifier.Push(msg)
				}

				if msg.emailMessage.enable {
					msg.emailMessage.notifier.Push(msg)
				}
			}
		}
	}()

	return pusherCustom
}
