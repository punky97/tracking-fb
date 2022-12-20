package team_distribution

import (
	"github.com/punky97/go-codebase/core/cache/local"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"crypto/md5"
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"sync"
	"time"
)

type ZapHookHandler struct {
	mu      sync.RWMutex
	Webhook string
	Members []string
}

var localCache = local.NewLocalCache(5000)
var defaultHook = &ZapHookHandler{}

func NewTeamDistributeHandler(webhook string, members []string) *ZapHookHandler {
	return &ZapHookHandler{
		Webhook: webhook,
		Members: members,
	}
}

func (h *ZapHookHandler) Handle(entry zapcore.Entry, fields map[string]interface{}, appName string) {
	if entry.Level == zapcore.ErrorLevel || entry.Level == zapcore.FatalLevel || entry.Level == zapcore.PanicLevel {
		h.handleLog(entry.Message, entry.Stack, fields, appName)
	}
}

func TeamDistribution(entry zapcore.Entry, appName string) {
	if entry.Level == zapcore.ErrorLevel || entry.Level == zapcore.FatalLevel || entry.Level == zapcore.PanicLevel {
		defaultHook.mu.Lock()
		defaultHook.Webhook = viper.GetString("reporter.notifier.custom.slack.webhook")
		defaultHook.Members = viper.GetStringSlice("reporter.notifier.custom.slack.members")
		defer defaultHook.mu.Unlock()

		defaultHook.handleLog(entry.Message, entry.Stack, nil, appName)
	}
}

func getVNTimezone() *time.Location {
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	return loc
}

func hashMessage(stackTrace string) string {
	md5Hashed := md5.Sum([]byte(stackTrace))
	return fmt.Sprintf("mk_%v", fmt.Sprintf("%x", md5Hashed))
}

func (h *ZapHookHandler) handleLog(message string, stack string, fields map[string]interface{}, appName string) {
	if len(h.Webhook) < 1 {
		logger.BkLog.Info("Missing slack webhook config")
		return
	}

	vnTimeZone := getVNTimezone()
	messageKey := hashMessage(stack)
	now := time.Now().In(vnTimeZone)

	// A error message just push once time to channel per day
	if ts, hit, _ := localCache.Get(messageKey); hit == local.CacheCodeHit {
		t := time.Unix(cast.ToInt64(ts), 0).In(vnTimeZone)

		// If this message was pushed today
		if t.Day() == now.Day() {
			return
		}
	}

	// Push alert message to slack
	slackNoti := notifier.NewSlackNotificationCustomWithMembers(
		h.Webhook,
		h.Members,
	)

	m := make([]map[string]string, len(fields))
	var idx = 0
	for k, v := range fields {
		m[idx] = map[string]string{
			"info": k,
			"desc": fmt.Sprintf("%v", v),
		}
		idx++
	}

	slackNoti.Notify(slackNoti.Format(constant.ReportLog{
		ReportType: constant.CriticalError,
		Priority:   constant.ReportAlert,
		Data: map[string]interface{}{
			"app":    appName,
			"desc":   message,
			"detail": stack,
			"source": config.GetHostName(),
			"extras": m,
		},
	}))

	// Mark message was pushed
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, vnTimeZone)
	localCache.Set(messageKey, now.Unix(), endOfDay.Sub(now))
}
