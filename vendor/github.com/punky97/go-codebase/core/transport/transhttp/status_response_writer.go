/**
 https://raw.githubusercontent.com/thoas/stats/master/recorder.go
 */
package transhttp

import (
	"bufio"
	"fmt"
	"github.com/urfave/negroni"
	"net"
	"net/http"
)

type ResponseWriter interface {
	negroni.ResponseWriter
}

type beforeFunc func(negroni.ResponseWriter)

type recorderResponseWriter struct {
	http.ResponseWriter
	status      int
	size        int
	beforeFuncs []beforeFunc
}

func NewRecorderResponseWriter(w http.ResponseWriter, statusCode int) negroni.ResponseWriter {
	return &recorderResponseWriter{w, statusCode, 0, nil}
}

func (r *recorderResponseWriter) WriteHeader(code int) {
	routerPattern := r.Header().Get(RoutePatternHeader)

	r.Header().Del(RoutePatternHeader)
	defer r.Header().Set(RoutePatternHeader, routerPattern)

	r.ResponseWriter.WriteHeader(code)
	r.status = code
}

func (r *recorderResponseWriter) Flush() {
	flusher, ok := r.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}

func (r *recorderResponseWriter) Status() int {
	return r.status
}

func (r *recorderResponseWriter) Write(b []byte) (int, error) {
	if !r.Written() {
		// The status will be StatusOK if WriteHeader has not been called yet
		r.WriteHeader(http.StatusOK)
	}
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

// Proxy method to Status to add support for gocraft
func (r *recorderResponseWriter) StatusCode() int {
	return r.Status()
}

func (r *recorderResponseWriter) Size() int {
	return r.size
}

func (r *recorderResponseWriter) Written() bool {
	return r.StatusCode() != 0
}

func (r *recorderResponseWriter) CloseNotify() <-chan bool {
	return r.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

func (r *recorderResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

func (r *recorderResponseWriter) Before(before func(negroni.ResponseWriter)) {
	r.beforeFuncs = append(r.beforeFuncs, before)
}
