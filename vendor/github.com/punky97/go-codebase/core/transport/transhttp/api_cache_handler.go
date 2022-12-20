package transhttp

import (
	"github.com/punky97/go-codebase/core/cache"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type ApiCacheValue struct {
	ContentType string
	Value       []byte
}

func GenKeyCache(appName string, uri string) string {
	return fmt.Sprintf("%v:cache:%v", appName, uri)
}

// ResponseRecorder is an implementation of http.ResponseWriter that
// records its mutations for later inspection in tests.
type ResponseCache struct {
	CacheKey  string
	Duration  int
	ApiMethod string
	Code      int
	w         http.ResponseWriter

	wroteHeader bool
}

// NewResponseCache returns an initialized ResponseCache.
func NewResponseCache(w http.ResponseWriter, cacheKey, apiMethod string, duration int) *ResponseCache {
	return &ResponseCache{
		CacheKey:  cacheKey,
		Duration:  duration,
		ApiMethod: apiMethod,
		w:         w,
		Code:      200,
	}
}

// Header returns the response headers.
func (rw *ResponseCache) Header() http.Header {
	return rw.w.Header()
}

func (rw *ResponseCache) writeHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.Code = code
}

// Write always succeeds and writes to rw.Body, if not nil.
func (rw *ResponseCache) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.writeHeader(http.StatusOK)
	}

	rw.setCache(buf)
	rw.w.Write(buf)

	return len(buf), nil
}

// WriteHeader sets rw.Code. After it is called, changing rw.Header
// will not affect rw.HeaderMap.
func (rw *ResponseCache) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.writeHeader(code)
}

func (rw *ResponseCache) setCache(data []byte) {
	contentType := rw.w.Header().Get("Content-Type")

	if rw.Code == http.StatusOK && !utils.IsStringSliceContains([]string{http.MethodDelete, http.MethodPost, http.MethodPut}, rw.ApiMethod) {
		keyCache := rw.CacheKey
		duration := rw.Duration

		if keyCache != "" && duration > 0 {
			acv := ApiCacheValue{
				ContentType: contentType,
				Value:       data,
			}
			cache.BkCache.SetCacheWithTimeout(keyCache, acv, duration)
		}
	}
}

type apiCacheHandler struct {
	handler  http.Handler
	appName  string
	duration int
}

func ApiCacheHandler(h http.Handler, appName string, duration int) http.Handler {
	return &apiCacheHandler{
		handler:  h,
		appName:  appName,
		duration: duration,
	}
}

func (h *apiCacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	keyCache := GenKeyCache(h.appName, r.RequestURI)
	if utils.IsStringSliceContains([]string{http.MethodDelete, http.MethodPut, http.MethodPost}, r.Method) {
		cache.BkCache.Delete(context.Background(), keyCache)
	} else {
		content, hit, err := cache.BkCache.GetCache(r.Context(), keyCache)
		if hit > 0 && err == nil {
			logger.BkLog.Debug("Cache Hit! \n", keyCache, content, hit)

			cacheData := &ApiCacheValue{}
			if cByte, err := json.Marshal(content); err == nil {
				err = json.Unmarshal(cByte, cacheData)
				if err == nil {
					RespondMessageWithContentType(w, http.StatusNotModified, string(cacheData.Value), cacheData.ContentType)
					return
				}
			}

		}
	}

	wrc := NewResponseCache(w, keyCache, r.Method, h.duration)
	h.handler.ServeHTTP(wrc, r)
}
