package utils

import (
	"github.com/punky97/go-codebase/core/logger"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Counter --
type Counter struct {
	Name    string `json:"name"`
	Counter int64  `json:"counter"`
}

// AddCounter --
func (c *Counter) AddCounter(delta int64) {
	atomic.AddInt64(&c.Counter, delta)
}

// CounterWatcher --
type CounterWatcher struct {
	mu         sync.RWMutex
	counter    []*Counter
	counterMap map[string]*Counter
}

// NewWatcher --
func NewWatcher(ctx context.Context) *CounterWatcher {
	w := &CounterWatcher{}

	w.counterMap = make(map[string]*Counter)

	go func() {
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				report := w.build()
				logger.BkLog.Info("Watcher: ", report)
			case <-ctx.Done():
				logger.BkLog.Info("Stop counter watcher")
				return
			}
		}
	}()

	return w
}

// GetReport --
func (w *CounterWatcher) GetReport() string {
	return w.build()
}

func (w *CounterWatcher) build() string {
	var result []string
	for _, c := range w.counter {
		b, err := json.Marshal(c)
		if err == nil {
			result = append(result, string(b))
		}
	}
	common := w.buildComon()
	result = append(result, common)
	return strings.Join(result, ",")
}
func (w *CounterWatcher) buildComon() string {
	var result []string
	// go routine
	result = append(result, marshal("routine", runtime.NumGoroutine()))
	result = append(result, marshal("num_cgo", runtime.NumGoroutine()))
	return strings.Join(result, ",")
}

func marshal(k string, v interface{}) string {
	b, err := json.Marshal(map[string]interface{}{"name": k, "value": runtime.NumGoroutine()})
	if err != nil {
		return ""
	}
	return string(b)
}

// MaxCounter --
const MaxCounter = 10000

// NewCounter --
func (w *CounterWatcher) NewCounter(name string) *Counter {
	return w.newCounter(getFullname(getCaller, name))
}

func getFullname(caller func() string, name string) string {
	return fmt.Sprintf("%v_%v", name, caller())
}

func (w *CounterWatcher) newCounter(fullName string) *Counter {
	if len(w.counter) > MaxCounter {
		logger.BkLog.Warn("Maximum number of counter has reached. Exiting")
		return nil
	}
	c := &Counter{fullName, 0}

	w.mu.Lock()
	w.counter = append(w.counter, c)
	w.counterMap[fullName] = c
	w.mu.Unlock()

	return c
}

// AddToCounter --
func (w *CounterWatcher) AddToCounter(name string, delta int64) {
	fullName := getFullname(getCaller, name)
	var ok bool
	var counter *Counter
	if counter, ok = w.counterMap[fullName]; !ok {
		if len(w.counter) > MaxCounter {
			logger.BkLog.Warn("Maximum number of counter has reached. Exiting. Cannot log %v %v", fullName, delta)
			return
		}
		counter = w.newCounter(fullName)
	}
	counter.AddCounter(delta)
}

func getCaller() string {
	pc, _, line, ok := runtime.Caller(3)
	var caller string
	if ok {
		details := runtime.FuncForPC(pc)
		caller = fmt.Sprintf("%v_%v", filepath.Base(details.Name()), line)
	}
	return caller
}
