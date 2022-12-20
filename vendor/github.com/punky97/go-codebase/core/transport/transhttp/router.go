package transhttp

import (
	"github.com/gorilla/mux"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"net/http"
	"time"
)

const (
	DefaultTimeout    = 10000 // ms
	DefaultDevTimeout = 10000 // ms
)

// Route -- Defines a single route, e.g. a human readable name, HTTP method,
// pattern the function that will execute when the route is called.
type Route struct {
	Name        string
	Method      string
	BasePath    string
	Pattern     string
	Handler     http.Handler
	Middlewares []negroni.Handler
	AuthInfo    AuthInfo
	Timeout     int64
	handler     http.Handler

	// For gen swagger document
	Tags                []string
	Description         string
	ResponseDescription string
	RequestDescription  string
	Deprecated          bool
	Summary             string
	ApiCache            bool
	ApiCacheDuration    int
}

// AuthInfo -- authentication and authorization for route
type AuthInfo struct {
	Enable         bool
	Middleware     negroni.Handler
	TokenType      string
	RequiredFields []string
	RestrictScopes []string
}

// Routes -- Defines the type Routes which is just an array (slice) of Route structs.
type Routes []Route

// NewRouter -- load all routers
func NewRouter(routes Routes, appName string, tMs int64, isLogDev bool) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	timeout := tMs
	if timeout <= 0 {
		timeout = viper.GetInt64("api.timeout")
	}
	if timeout <= 0 {
		if isLogDev {
			timeout = DefaultDevTimeout
		} else {
			timeout = DefaultTimeout
		}
	}

	logger.BkLog.Infof("Config Timeout API: %v", timeout)

	for _, route := range routes {
		handlers := make([]negroni.Handler, 0)
		if route.AuthInfo.Middleware != nil {
			handlers = append(handlers, route.AuthInfo.Middleware)
		}
		for _, mid := range route.Middlewares {
			handlers = append(handlers, mid)
		}

		rh := route.Handler

		if route.Timeout >= 0 {
			if route.Timeout > 0 {
				timeout = route.Timeout
			}

			routePattern := route.Method + " " + route.BasePath + ":" + route.Pattern
			serviceName := appName
			if config.GetHostName() != "" {
				serviceName = config.GetHostName()
			}

			rh = LogTimeoutHandler(route.Handler, serviceName, routePattern, time.Duration(timeout)*time.Millisecond)
		}

		if route.ApiCache {
			rh = ApiCacheHandler(rh, appName, route.ApiCacheDuration)
		}

		handlers = append(handlers, negroni.Wrap(rh))

		router.
			Methods(route.Method).
			Path(route.BasePath + route.Pattern).
			Handler(negroni.New(handlers...))
	}

	return router
}

func RegisterHealthCheck(r http.Handler, svcName string) *http.ServeMux {
	mixMux := http.NewServeMux()
	mixMux.Handle("/", r)
	mixMux.Handle("/service-health", http.TimeoutHandler(&healthCheckHandler{Service: svcName}, time.Second, "API Timeout"))

	return mixMux
}

type healthCheckHandler struct {
	Service string
}

func (h *healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if svc := r.URL.Query().Get("svc"); svc == "" {
		// TODO: return StatusGone for this case
		logger.BkLog.Warnf("Health Check request is missing service name")
	} else if svc != h.Service {
		logger.BkLog.Warnf("Wrong service name actual %v, expect %v", svc, h.Service)
		RespondMessage(w, http.StatusGone, "conflict service name")
		return
	}
	RespondMessage(w, http.StatusOK, "active")
}
