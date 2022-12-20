package transhttp

import (
	"bytes"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type timeoutHandler struct {
	handler http.Handler
	body    string
	dt      time.Duration
	app     string
}

func BkTimeoutHandler(app string, h http.Handler, dt time.Duration, msg string) http.Handler {
	return &timeoutHandler{
		handler: h,
		body:    msg,
		dt:      dt,
		app:     app,
	}
}

// ErrHandlerTimeout is returned on ResponseWriter Write calls
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("http: Handler timeout")

func (h *timeoutHandler) errorBody() string {
	if h.body != "" {
		return h.body
	}
	return "<html><head><title>Timeout</title></head><body><h1>Timeout</h1></body></html>"
}

func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancelCtx := context.WithTimeout(r.Context(), h.dt)
	defer cancelCtx()

	r = r.WithContext(ctx)
	done := make(chan struct{})
	start := time.Now()
	initDeadline, _ := ctx.Deadline()
	tw := &timeoutWriter{
		w: w,
		h: make(http.Header),
	}
	panicChan := make(chan interface{}, 1)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				h.dumpPanic(p)
				panicChan <- p
			}
		}()
		h.handler.ServeHTTP(tw, r)
		close(done)
	}()
	select {
	case p := <-panicChan:
		panic(p)
	case <-done:
		tw.mu.Lock()
		defer tw.mu.Unlock()
		dst := w.Header()
		for k, vv := range tw.h {
			dst[k] = vv
		}
		if !tw.wroteHeader {
			tw.code = http.StatusOK
		}
		w.WriteHeader(tw.code)
		w.Write(tw.wbuf.Bytes())
	case <-ctx.Done():
		tw.mu.Lock()
		defer tw.mu.Unlock()
		deadline, _ := ctx.Deadline()
		logger.BkLog.Debug("Timeout from: ", start, " to ", time.Now(), ": ", initDeadline, " - ", deadline)
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, h.errorBody())
		tw.timedOut = true
		return
	}
}

func (h *timeoutHandler) dumpPanic(err interface{}) {
	logger.BkLog.Errorw(fmt.Sprintf("Recovery catch router panic: %v", err))

	var criticalNotifier = notifier.CriticalNotifier()
	stack := make([]byte, 1024*12)
	stack = stack[:runtime.Stack(stack, false)]

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Panic when notify: %v", r))
			}
		}()

		var detail = fmt.Sprintf("%v", err)
		if len(stack) > 0 {
			detail = fmt.Sprintf("%v", string(stack))
		}

		report := map[string]interface{}{
			"app":    h.app,
			"desc":   fmt.Sprintf("Panic happened: %v", err),
			"detail": detail,
			"extras": []map[string]string{},
			"source": h.app,
		}

		criticalNotifier.Notify(criticalNotifier.Format(constant.ReportLog{
			ReportType: constant.CriticalError,
			Priority:   constant.ReportAlert,
			Data:       report,
		}))
	}()
}

type timeoutWriter struct {
	w    http.ResponseWriter
	h    http.Header
	wbuf bytes.Buffer

	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
	code        int
}

func (tw *timeoutWriter) Header() http.Header { return tw.h }

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.writeHeader(http.StatusOK)
	}
	return tw.wbuf.Write(p)
}

func (tw *timeoutWriter) WriteHeader(code int) {
	checkWriteHeaderCode(code)
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.writeHeader(code)
}

func (tw *timeoutWriter) writeHeader(code int) {
	tw.wroteHeader = true
	tw.code = code
}

func checkWriteHeaderCode(code int) {
	// Issue 22880: require valid WriteHeader status codes.
	// For now we only enforce that it's three digits.
	// In the future we might block things over 599 (600 and above aren't defined
	// at http://httpwg.org/specs/rfc7231.html#status.codes)
	// and we might block under 200 (once we have more mature 1xx support).
	// But for now any three digits.
	//
	// We used to send "HTTP/1.1 000 0" on the wire in responses but there's
	// no equivalent bogus thing we can realistically send in HTTP/2,
	// so we'll consistently panic instead and help people find their bugs
	// early. (We can't return an error from WriteHeader even if we wanted to.)
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}
