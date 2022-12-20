package transhttp

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	TimeoutMessage = "Request Timeout"

	RoutePatternHeader = "X-Route-Pattern"
)

type logTimeoutHandler struct {
	appName       string
	handler       http.Handler
	originHandler http.Handler
	routePattern  string
	timeout       time.Duration
}

func LogTimeoutHandler(h http.Handler, appName string, routePattern string, t time.Duration) http.Handler {
	return &logTimeoutHandler{
		appName:       appName,
		handler:       BkTimeoutHandler(appName, h, t, TimeoutMessage),
		originHandler: h,
		routePattern:  routePattern,
		timeout:       t,
	}
}

func (h *logTimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if IsWebSocket(r) {
		h.originHandler.ServeHTTP(w, r)
	} else {
		lrw := NewLoggingResponseWriter(w)
		// write X-Route-Pattern
		lrw.Header().Set(RoutePatternHeader, h.routePattern)
		startTime := time.Now()

		h.handler.ServeHTTP(lrw, r)

		if lrw.statusCode == http.StatusServiceUnavailable && time.Now().Sub(startTime) > h.timeout {
			// Try to get the real IP
			remoteAddr := r.RemoteAddr
			if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
				remoteAddr = realIP
			}

			// get address only
			if len(strings.Split(remoteAddr, ":")) == 2 {
				remoteAddr = strings.Split(remoteAddr, ":")[0]
			}

			reportLog := &constant.ReportLog{
				ReportType: constant.APITimeoutReport,
				Priority:   constant.ReportAlert,
				Data: map[string]interface{}{
					"service": h.appName,
					"method":  r.Method,
					"source":  h.appName,
					"uri":     r.RequestURI,
					"remote":  remoteAddr,
					"timeout": h.timeout.Nanoseconds(),
				},
			}

			// dispatch notify
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Panic when notify: %v", r))
					}
				}()

				slackNoti := notifier.DefaultNotifier()
				logBytes, err := json.Marshal(reportLog)
				if err == nil {
					slackNoti.Notify(slackNoti.Format(*reportLog))
					logger.BkLog.Warnf("api timeout from %v: %v", startTime, string(logBytes))
				} else {
					logger.BkLog.Errorw(fmt.Sprintf("Error when report log: %v", err), "report", reportLog)
				}
			}()
		}
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	// WriteHeader(int) is not called if our response implicitly returns 200 OK, so
	// we default to that status code.
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
