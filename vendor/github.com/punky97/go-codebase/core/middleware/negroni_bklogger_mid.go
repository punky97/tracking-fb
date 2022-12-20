package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type timer interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

type realClock struct{}

func (rc *realClock) Now() time.Time {
	return time.Now()
}

func (rc *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

type LogConfig struct {
	isLogDetail     bool
	isLogBody       bool
	isOnlyTextBased bool
	textBaseTypes   []string
	isLogBodyAnyway bool
	ignoreHeaders   []string
	ignorePatterns  *regexp.Regexp
}

var logConfig *LogConfig

// NegroniBkLoggerMid is a middleware handler that logs the request as it goes in and the response as it goes out.
type NegroniBkLoggerMid struct {
	// Logger is the log.Logger instance used to log messages with the Logger middleware
	HttpTracingHeader string
	DMSTracingHeader  string
	// Name is the name of the application as recorded in latency metrics
	Name        string
	Before      func(*logger.BkLogger, *http.Request, string, string)
	After       func(*logger.BkLogger, negroni.ResponseWriter, time.Duration, string, string)
	logStarting bool

	clock timer

	// Exclude URLs from logging
	excludeURLs []string
}

// NewNegroniBkLoggerMidFromLogger returns a new *Middleware which writes to a given logrus logger.
func NewNegroniBkLoggerMidFromLogger(name string, httpTracingHeader, dmsTracingHeader string) *NegroniBkLoggerMid {
	if logConfig == nil {
		StartLoadLogConfig()
	}

	return &NegroniBkLoggerMid{
		HttpTracingHeader: httpTracingHeader,
		DMSTracingHeader:  dmsTracingHeader,
		Name:              name,
		Before:            DefaultBefore,
		After:             DefaultAfter,
		logStarting:       true,
		clock:             &realClock{},
	}
}

func StartLoadLogConfig() {
	var duration = time.Duration(viper.GetInt("api.logger.check_config_interval"))

	if duration <= 0 {
		duration = time.Duration(viper.GetInt("logger.check_config_interval"))
		if duration <= 0 {
			duration = 30
		}
	}
	var ticker = time.NewTicker(duration * time.Second)
	logConfig = LoadLogConfig()
	go func() {
		for {
			select {
			case <-ticker.C:
				temp := LoadLogConfig()
				logConfig = temp
			}
		}
	}()
}

func LoadLogConfig() *LogConfig {
	var isLogDetail = viper.GetBool("api.logger.log_detail")
	if !isLogDetail {
		isLogDetail = viper.GetBool("logger.log_detail")
	}

	var isLogBody = viper.GetBool("api.logger.log_body")
	if !isLogBody {
		isLogBody = viper.GetBool("logger.log_body")
	}

	var isOnlyTextBased = viper.GetBool("api.logger.only_text_based_body")
	if !isOnlyTextBased {
		isOnlyTextBased = viper.GetBool("logger.only_text_based_body")
	}

	var textBaseTypes = viper.GetStringSlice("api.logger.text_based_content_types")
	if len(textBaseTypes) == 0 {
		textBaseTypes = viper.GetStringSlice("logger.text_based_content_types")
	}

	var isLogBodyAnyway = viper.GetBool("api.logger.log_body_anyway")
	if !isLogBodyAnyway {
		isLogBodyAnyway = viper.GetBool("logger.log_body_anyway")
	}

	var ignoreHeaders = viper.GetStringSlice("api.logger.ignore_headers")
	var ignorePatternStr = viper.GetString("api.logger.ignore_patterns")
	var rg *regexp.Regexp
	if ignorePatternStr != "" {
		var err error
		rg, err = regexp.Compile(ignorePatternStr)
		if err != nil {
			logger.BkLog.Warnf("Error when compile pattern %v: %v", ignorePatternStr, err)
			rg = nil
		}
	}

	return &LogConfig{
		isLogDetail:     isLogDetail,
		isLogBody:       isLogBody,
		isOnlyTextBased: isOnlyTextBased,
		textBaseTypes:   textBaseTypes,
		isLogBodyAnyway: isLogBodyAnyway,
		ignoreHeaders:   ignoreHeaders,
		ignorePatterns:  rg,
	}
}

// SetLogStarting accepts a bool to control the logging of "started handling
// request" prior to passing to the next middleware
func (m *NegroniBkLoggerMid) SetLogStarting(v bool) {
	m.logStarting = v
}

// ExcludeURL adds a new URL u to be ignored during logging. The URL u is parsed, hence the returned error
func (m *NegroniBkLoggerMid) ExcludeURL(u string) error {
	if _, err := url.Parse(u); err != nil {
		return err
	}
	m.excludeURLs = append(m.excludeURLs, u)
	return nil
}

var defaultExcludeURL = []string{"/service-health"}

// ExcludedURLs returns the list of excluded URLs for this middleware
func (m *NegroniBkLoggerMid) ExcludedURLs() []string {
	if len(m.excludeURLs) > 0 {
		return m.excludeURLs
	} else {
		return defaultExcludeURL
	}
}

func (m *NegroniBkLoggerMid) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if m.Before == nil {
		m.Before = DefaultBefore
	}

	if m.After == nil {
		m.After = DefaultAfter
	}

	for _, u := range m.ExcludedURLs() {
		if r.URL.Path == u {
			next(rw, r)
			return
		}
	}

	start := m.clock.Now()

	// Try to get the real IP
	remoteAddr := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		remoteAddr = realIP
	}

	// add logger to ctx
	tracingId := r.Header.Get(m.HttpTracingHeader)

	if tracingId == "" {
		seq := ksuid.Sequence{Seed: ksuid.New()}
		id, err := seq.Next()
		if err == nil {
			tracingId = id.String()
		} else {
			logger.BkLog.Warn("cant gen random uuid: ", err)
		}
	}

	ctx := logger.AddLogCtx(r.Context(), *logger.BkLog, tracingId)
	ctx = context.WithValue(ctx, m.DMSTracingHeader, tracingId)

	l := logger.LoggerCtx(ctx)
	if m.logStarting {
		m.Before(l, r, remoteAddr, fmt.Sprintf("Started handling api request %v", r.RequestURI))
	}

	next(rw, r.WithContext(ctx))
	latency := m.clock.Since(start)
	res := rw.(negroni.ResponseWriter)

	m.After(l, res, latency, m.Name, fmt.Sprintf("Completed handling api request %v", r.RequestURI))
}

// BeforeFunc is the func type used to modify or replace the *logrus.Entry prior
// to calling the next func in the middleware chain
type BeforeFunc func(*logger.BkLogger, *http.Request, string, string)

// AfterFunc is the func type used to modify or replace the *logrus.Entry after
// calling the next func in the middleware chain
type AfterFunc func(*logger.BkLogger, negroni.ResponseWriter, time.Duration, string, string)

// DefaultBefore is the default func assigned to *Middleware.Before
func DefaultBefore(l *logger.BkLogger, req *http.Request, remoteAddr, msg string) {
	hostname := config.GetHostName()
	isLogBodyEnable := logConfig.isLogDetail
	if isLogBodyEnable && logConfig.ignorePatterns != nil {
		if logConfig.ignorePatterns.MatchString(req.URL.Path) {
			isLogBodyEnable = false
		}
	}

	if !isLogBodyEnable {
		l.Infow(msg,
			"request", req.RequestURI,
			"method", req.Method,
			"remote", remoteAddr,
			"host", hostname,
		)
		return
	}

	header := utils.DumpHeaderMap(req, logConfig.ignoreHeaders...)
	if !logConfig.isLogBody {
		l.Infow(msg,
			"request", req.RequestURI,
			"method", req.Method,
			"remote", remoteAddr,
			"header", header,
			"host", hostname,
		)
		return
	}
	// Process to log body
	contentType := req.Header.Get("Content-Type")
	var body string
	var isBodySet bool
	if logConfig.isOnlyTextBased ||
		(contentType != "" && isTextBaseBody(contentType)) {
		buf, _ := ioutil.ReadAll(req.Body)
		body = string(buf)
		isBodySet = true
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	}

	if !isBodySet && logConfig.isLogBodyAnyway {
		buf, _ := ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		body = base64.StdEncoding.EncodeToString(buf)
		isBodySet = true
	}
	if isBodySet {
		l.Infow(msg,
			"request", req.RequestURI,
			"method", req.Method,
			"remote", remoteAddr,
			"header", header,
			"body", body,
			"host", hostname,
		)
	} else {
		l.Infow(msg,
			"request", req.RequestURI,
			"method", req.Method,
			"remote", remoteAddr,
			"header", header,
			"host", hostname,
		)
	}
}

func isTextBaseBody(s string) bool {
	types := logConfig.textBaseTypes
	for _, contentType := range types {
		if strings.HasPrefix(s, contentType) {
			return true
		}
	}
	return false
}

// DefaultAfter is the default func assigned to *Middleware.After
func DefaultAfter(l *logger.BkLogger, res negroni.ResponseWriter, latency time.Duration, name, msg string) {
	l.Infow(msg,
		"status", res.Status(),
		"text_status", http.StatusText(res.Status()),
		"took", latency,
		"host", config.GetHostName(),
	)
}

type GatewayLoggerMid struct {
	HttpTracingHeader string
}

func NewGatewayLoggerMid(httpTracingHeader string) *GatewayLoggerMid {
	return &GatewayLoggerMid{
		HttpTracingHeader: httpTracingHeader,
	}
}

func (m *GatewayLoggerMid) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// add logger to ctx
	tracingId := r.Header.Get(m.HttpTracingHeader)

	if tracingId == "" {
		seq := ksuid.Sequence{Seed: ksuid.New()}
		id, err := seq.Next()
		if err == nil {
			tracingId = id.String()
		} else {
			logger.BkLog.Warn("cant gen random uuid: ", err)
		}
	}

	ctx := logger.AddLogCtx(r.Context(), *logger.BkLog, tracingId)

	next(w, r.WithContext(ctx))
}
