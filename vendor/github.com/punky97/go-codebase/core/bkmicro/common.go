package bkmicro

import (
	"github.com/punky97/go-codebase/core/bkgrpc/db_tools"
	"github.com/punky97/go-codebase/core/bkgrpc/dialer"
	"github.com/punky97/go-codebase/core/bkgrpc/grpcinterceptor"
	"github.com/punky97/go-codebase/core/bkgrpc/healthcheck"
	"github.com/punky97/go-codebase/core/bkgrpc/registry"
	"github.com/punky97/go-codebase/core/bkgrpc/reporter"
	"github.com/punky97/go-codebase/core/bkgrpc/timeout"
	"github.com/punky97/go-codebase/core/bkgrpc/tracing"
	"github.com/punky97/go-codebase/core/bkmigration"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/debug"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/middleware"
	reporter2 "github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

const (
	DefaultDmsTimeout    = 10000 //ms
	DefaultDevDmsTimeout = 10000 //ms
)

var (
	apiReporterMiddlerware *middleware.NegroniBkStatsMid
)

type GrpcInterceptorOpts struct {
	Stream grpc.StreamServerInterceptor
	Unary  grpc.UnaryServerInterceptor
}

// InitCommon -- initializes common things: logger, config, service discovery, tracing
func (app *App) InitCommon() {
	app.PreCheck()
	app.InitLogger()
	app.InitConfig()
	app.InitSdConnection()
	app.setMaxGR()
	app.initAdminParams()
	app.InitDebug()
	app.InitAlert()
	app.InitDirtyFieldChecking()
	app.InitLimitRecordAndFieldSelect()
	app.InitApiCache()
}

// OnCloseCallback --
type OnCloseCallback func(*App)

// NewDMSAppTest returns new app only for test
func NewDMSAppTest(name, version string, onClose OnCloseCallback) *App {
	app := App{
		Name:    name,
		Version: version,
		Type:    BkAppTypeDMS,
	}
	// init default flags, then parse
	app.InitDefaultFlags()
	app.ParseFlags()
	// init for bkMicroApp
	// app.InitCommon()

	logger.BkLog.Infof("%v. Starting...", app.Info())

	// handle sigterm
	utils.HandleSigterm(func() {
		logger.BkLog.Infof("%v. Stopping...", app.Info())
		onClose(&app)

		debug.StopProfiling()
		notifier.Close()
	})

	return &app
}

func (app *App) initAdminParams() {
	cleanup := viper.GetBool("admin.service.registry.cleanup_at_start")
	if !app.CleanupAtStart && cleanup {
		app.CleanupAtStart = cleanup
	}
}

func (app *App) setMaxGR() {
	max := viper.GetInt("root.max_procs")
	if max > 0 {
		runtime.GOMAXPROCS(max)
	}

}

// GetNewGRPCServer -- define common grpc middleware, return
func (app *App) GetNewGRPCServer(interceptors ...GrpcInterceptorOpts) *grpc.Server {
	isDebugMode := viper.GetBool("debug_mode.enable")
	// init global middleware
	opts := make([]grpc.ServerOption, 0)

	// stream interceptors
	var streamInterceptors = make([]grpc.StreamServerInterceptor, 0)
	streamInterceptors = append(streamInterceptors, grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(grpcPanicHandler(app.DisplayName, config.GetHostName()))))
	streamInterceptors = append(streamInterceptors, grpcinterceptor.BkContextServerStreamInterceptor())

	for _, intercept := range interceptors {
		if intercept.Stream != nil {
			streamInterceptors = append(streamInterceptors, intercept.Stream)
		}
	}

	opts = append(opts, grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(streamInterceptors...),
	))

	// unary interceptors
	unaryInterceptors := make([]grpc.UnaryServerInterceptor, 0)

	dmsLogEnable := viper.GetBool("dms.enable_trace_log")
	if dmsLogEnable {
		logger.BkLog.Infof("Enable dms trace log")
		defaultThreshold := viper.GetInt("dms_trace.threshold_ms")
		if defaultThreshold <= 0 {
			defaultThreshold = 1000
		}

		filterMethods := viper.GetStringSlice("dms_trace.filter_methods")
		alertFilters := make([]grpcinterceptor.AlertFilter, 0)
		for _, method := range filterMethods {
			threshold := viper.GetInt(fmt.Sprintf("dms_trace.threshold.%s", method))
			if threshold > 0 {
				alertFilters = append(alertFilters, grpcinterceptor.AlertFilter{
					Method:    method,
					Threshold: time.Duration(threshold) * time.Millisecond,
				})
			}
		}

		unaryInterceptors = append(unaryInterceptors, grpcinterceptor.BkLoggerServerInterceptor(alertFilters, time.Duration(defaultThreshold)*time.Millisecond, isDebugMode))
	}

	if reporter2.IsMetricsEnable() {
		unaryInterceptors = append(unaryInterceptors, reporter.UnaryServerInterceptor(fmt.Sprintf("dms-%v", app.DisplayName)))
	}

	unaryInterceptors = append(unaryInterceptors, grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(grpcPanicHandler(app.DisplayName, config.GetHostName()))))

	unaryInterceptors = append(unaryInterceptors, grpcinterceptor.BkContextServerInterceptor())

	hostname := config.GetHostName()
	unaryInterceptors = append(unaryInterceptors, timeout.UnaryServerInterceptor(hostname))

	for _, intercept := range interceptors {
		if intercept.Unary != nil {
			unaryInterceptors = append(unaryInterceptors, intercept.Unary)
		}
	}

	opts = append(opts, grpc.UnaryInterceptor(
		grpc_middleware.ChainUnaryServer(unaryInterceptors...),
	))
	// create new server
	srv := grpc.NewServer(opts...)

	// register healthcheck server
	healthcheck.RegisterHealthCheckServer(srv, app.DisplayName)

	reflection.Register(srv)

	return srv
}

func grpcPanicHandler(service string, app string) func(interface{}) error {
	var criticalNotifier = notifier.CriticalNotifier()

	return func(p interface{}) error {
		logger.BkLog.Errorw(fmt.Sprintf("Recovery catch grpc panic: %v", p))
		stack := make([]byte, 1024*12)
		stack = stack[:runtime.Stack(stack, false)]

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Panic when notify: %v", r))
				}
			}()

			var detail = fmt.Sprintf("%v", p)
			if len(stack) > 0 {
				detail = fmt.Sprintf("%v", string(stack))
			}

			report := map[string]interface{}{
				"app":    service,
				"desc":   fmt.Sprintf("Panic happened: %v", p),
				"detail": detail,
				"extras": []map[string]string{},
				"source": app,
			}

			criticalNotifier.Notify(criticalNotifier.Format(constant.ReportLog{
				ReportType: constant.CriticalError,
				Priority:   constant.ReportAlert,
				Data:       report,
			}))
		}()

		return grpc.Errorf(codes.Internal, "%s", p)
	}
}

// Shutdown - send data to Exit channel to unlock the handleExit function
func (app *App) Shutdown() {
	if app.IsRunning {
		app.Exit <- true
	}
}

// WaitForExit - wait for data from Exit channel for 5 seconds
func (app *App) WaitForExit() {
	select {
	case <-app.Exit:
	case <-time.After(15 * time.Second):
		logger.BkLog.Warn("Cannot receive data from Exit channel after 15 seconds")
	}
}

// NewDMSApp returns new app
func NewDMSApp(name, version string, onClose OnCloseCallback) (*App, *grpc.Server) {
	app := App{
		Name:    name,
		Version: version,
		Type:    BkAppTypeDMS,
		Exit:    make(chan bool),
	}
	// init default flags, then parse
	app.InitDefaultFlags()
	app.ParseFlags()
	// init for bkMicroApp
	app.InitCommon()
	if viper.GetBool("migration.auto_migration") {
		logger.BkLog.Infof("%v was enabled auto migration. Starting find and apply migrations are pending...", app.Name)
		if err := bkmigration.RunAutoMigration(); err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("[DMS] %s has run auto migration fail with err %s", app.DisplayName, err))
			slackNotifier := notifier.DefaultNotifier()
			slackNotifier.Notify(slackNotifier.Format(constant.ReportLog{
				ReportType: constant.ConsumerErrorReport,
				Priority:   constant.ReportAlert,
				Data: map[string]interface{}{
					"dms":      name,
					"version":  version,
					"messages": err,
				},
			}))
			panic(err)
			return nil, nil
		}
	}
	logger.BkLog.Infof("%v. Starting...", app.Info())

	srv := app.GetNewGRPCServer()

	// handle sigterm
	utils.HandleSigterm(func() {
		logger.BkLog.Infof("%v. Stopping...", app.Info())
		reporter.StopServerCollector()
		if app.IsRunning {
			srv.GracefulStop()
		} else {
			onClose(&app)
		}
		debug.StopProfiling()
		notifier.Close()
		if app.IsRunning {
			app.WaitForExit()
		}
	})

	return &app, srv
}

// RunDMS -- run DMS server
func (app *App) RunDMS(srv *grpc.Server) (err error) {
	address := fmt.Sprintf("%s:%d", app.GrpcHost, app.GrpcPort)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return
	}

	reporter.StartServerCollector()

	logger.BkLog.Infof("Starting gRPC service %v at %v", app.Name, address)

	app.IsRunning = true
	err = srv.Serve(lis)
	return
}

// NewAPIApp returns new app
func NewAPIApp(name, httpBasePath, version string, onClose OnCloseCallback) (*App, *http.Server) {
	app := App{
		Name:         name,
		Version:      version,
		Type:         BkAppTypeAPI,
		HTTPBasePath: httpBasePath,
		Exit:         make(chan bool),
	}
	// init default flags, then parse
	app.InitDefaultFlags()
	app.ParseFlags()
	// init for bkMicroApp
	app.InitCommon()

	logger.BkLog.Infof("%v. Starting...", app.Info())

	address := fmt.Sprintf("%v:%v", app.HTTPHost, app.HTTPPort)
	hs := &http.Server{
		Addr: address,
		// Note: when using ABL on AWS https://docs.aws.amazon.com/elasticloadbalancing/latest/application/application-load-balancers.html#connection-idle-timeout
		// IdleTimeout must be greater than ALB Idle Timeout
		IdleTimeout:  70 * time.Second,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 70 * time.Second,
	}

	// handle sigterm
	utils.HandleSigterm(func() {
		logger.BkLog.Infof("%v. Stopping...", app.Info())
		ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)
		if app.IsRunning {
			err := hs.Shutdown(ctx)
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("An error occured when http server is being shutdown, details: %v", err))
			}
		} else {
			onClose(&app)
		}
		if apiReporterMiddlerware != nil {
			apiReporterMiddlerware.Close()
		}
		debug.StopProfiling()
		notifier.Close()
		if app.IsRunning {
			app.WaitForExit()
		}
	})

	return &app, hs
}

// RunAPI -- run API server
func (app *App) RunAPI(hs *http.Server, mids ...negroni.Handler) (err error) {
	// init middlewares
	mr := transhttp.NewRouter(app.Routes, app.DisplayName, app.DefaultAPITimeout, app.LoggerDev)

	for _, m := range app.Middlewares {
		mids = append(mids, m)
	}

	n := app.InitGlobalAPIMiddlewares(mr, mids...)

	// add api health-check pod
	n2 := transhttp.RegisterHealthCheck(n, app.DisplayName)

	hs.Handler = n2

	// ListenAndServe
	hs.SetKeepAlivesEnabled(true)
	logger.BkLog.Infof("Starting HTTP service %v at %v", app.Name, hs.Addr)
	app.IsRunning = true
	err = hs.ListenAndServe()
	if err == http.ErrServerClosed {
		logger.BkLog.Infof("Server closed")
		err = nil
	}
	return
}

// InitGlobalAPIMiddlewares --
func (app *App) InitGlobalAPIMiddlewares(handler http.Handler, mids ...negroni.Handler) *negroni.Negroni {
	// using negroni mids for all requests
	n := negroni.New()

	logDisabled := viper.GetBool("api.disable_trace_log")

	if logDisabled {
		logger.BkLog.Info("Disabled request & response log middleware")
	} else {
		// use logger, print each incoming request and response
		n.Use(middleware.NewNegroniBkLoggerMidFromLogger("negroni", app.HttpTracingHeader, app.DmsTracingHeader))
	}

	if reporter2.IsMetricsEnable() {
		apiReporterMiddlerware = middleware.NewNegroniBkStatsMid(app.DisplayName)
		n.Use(apiReporterMiddlerware)

		apiReporterMiddlerware.Start()
	}

	// use recovery, catches panics and responds with a 500 response code
	n.Use(middleware.NewBkRecovery())

	// more mids
	if len(mids) > 0 {
		for _, mid := range mids {
			n.Use(mid)
		}
	}

	n.UseHandler(handler)

	return n
}

// InitGRPCClients -- init connections to dms services
func (app *App) InitGRPCClients(dmsCodeNames ...string) map[string]*grpc.ClientConn {
	logger.BkLog.Info("InitGRPCClients")
	m := make(map[string]*grpc.ClientConn)
	consul, err := registry.NewClient(app.SdAddress)
	if err != nil {
		logger.BkLog.Error(err)
		os.Exit(1)
	}

	for _, codename := range dmsCodeNames {
		dialopts := make([]dialer.DialOption, 0)
		interceptors := make([]grpc.UnaryClientInterceptor, 0)
		streamInterceptors := make([]grpc.StreamClientInterceptor, 0)
		// dial app dms
		if app.AllowLoopbackIP {
			dialopts = []dialer.DialOption{
				dialer.WithLocalBalancer(consul.Client, app.AllowLoopbackIP),
			}
		} else {
			dialopts = []dialer.DialOption{
				dialer.WithBalancer(consul.Client),
			}
		}

		// add tracing
		interceptors = append(interceptors, tracing.UnaryClientInterceptor(app.DmsTracingHeader))
		streamInterceptors = append(streamInterceptors, tracing.StreamClientInterceptor(app.DmsTracingHeader))

		// add limit field
		interceptors = append(interceptors, db_tools.UnaryClientInterceptor(app.DefaultLimitEnable, app.DefaultFieldSelect))
		streamInterceptors = append(streamInterceptors, db_tools.StreamClientInterceptor(app.DefaultLimitEnable, app.DefaultFieldSelect))

		dialopts = append(dialopts, dialer.WithBackoff())
		timeout := app.DefaultDMSTimeout
		if timeout <= 0 {
			timeout = viper.GetInt64("dms.timeout")
		}
		if timeout <= 0 {
			if app.LoggerDev {
				timeout = DefaultDevDmsTimeout
			} else {
				timeout = DefaultDmsTimeout
			}
		}

		logger.BkLog.Infof("Config Timeout DMS: %v", timeout)

		serviceName := app.DisplayName
		if config.GetHostName() != "" {
			serviceName = config.GetHostName()
		}

		defaultRetry := uint(viper.GetInt("dms.default_retry"))
		interceptors = append(interceptors, dialer.WithTimeout(serviceName, time.Duration(timeout)*time.Millisecond, defaultRetry))

		blockUntilConnection := viper.GetBool("admin.service.wait_dms_conn")

		if len(interceptors) > 0 {
			interceptorOpt := func(name string) (grpc.DialOption, error) {
				return grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(interceptors...)), nil
			}

			dialopts = append(dialopts, interceptorOpt)
		}
		if len(streamInterceptors) > 0 {
			streamInterceptorOpt := func(name string) (grpc.DialOption, error) {
				return grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(streamInterceptors...)), nil
			}
			dialopts = append(dialopts, streamInterceptorOpt)
		}

		cc, err := dialer.Dial(blockUntilConnection,
			BuildGRPCRegistryServiceName(
				DefaultGRPCNamespace,
				codename,
			),
			dialopts...,
		)

		if err != nil {
			logger.BkLog.Warnf("dialer to dms %v fail: %v", codename, err)
		} else {
			m[codename] = cc
			logger.BkLog.Infof("dialer to dms %v success, current dms state is %v", codename, cc.GetState())
		}
	}

	return m
}

// NewAPIGatewayApp -- create new API Gateway app
func NewAPIGatewayApp(name, version string, onClose OnCloseCallback) (*App, *http.Server) {
	app := App{
		Name:    name,
		Version: version,
		Type:    BkAppTypeAPIGW,
		Exit:    make(chan bool),
	}
	// init default flags, then parse
	app.InitDefaultFlags()
	app.ParseFlags()
	// init for bkMicroApp
	app.InitCommon()
	app.InitJWT()

	logger.BkLog.Infof("%v. Starting...", app.Info())

	addr := fmt.Sprintf("%v:%v", app.HTTPHost, app.HTTPPort)
	hs := &http.Server{
		Addr: addr,
		// Note: when using ABL on AWS https://docs.aws.amazon.com/elasticloadbalancing/latest/application/application-load-balancers.html#connection-idle-timeout
		// IdleTimeout must be greater than ALB Idle Timeout
		IdleTimeout:  70 * time.Second,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 70 * time.Second,
	}

	utils.HandleSigterm(func() {
		logger.BkLog.Infof("%v. Stopping...", app.Info())
		if app.IsRunning {
			hs.Shutdown(context.Background())
		} else {
			onClose(&app)
		}
		debug.StopProfiling()
		notifier.Close()
		if app.IsRunning {
			app.WaitForExit()
		}
	})

	return &app, hs
}

func (app *App) RunAPIGateway(hs *http.Server, r *mux.Router, middles ...negroni.Handler) (err error) {
	// init global middlewares
	n := app.initApiGatewayMiddlewares(r, middles...)
	n2 := transhttp.RegisterHealthCheck(n, app.DisplayName)
	hs.Handler = n2
	// Listen and serve
	hs.SetKeepAlivesEnabled(true)
	logger.BkLog.Infof("Starting API-GW HTTP service at %v", hs.Addr)

	app.IsRunning = true
	err = hs.ListenAndServe()
	return
}

func (app *App) initApiGatewayMiddlewares(handler http.Handler, middlewares ...negroni.Handler) *negroni.Negroni {
	// using negroni mids for all requests
	n := negroni.New()

	for _, m := range middlewares {
		n.Use(m)
	}

	newGatewayLoggerMid := middleware.NewGatewayLoggerMid(app.HttpTracingHeader)
	if newGatewayLoggerMid != nil {
		n.Use(newGatewayLoggerMid)
	} else {
		logger.BkLog.Error("Unable to initialize gateway logger.")
	}

	if reporter2.IsMetricsEnable() {
		apiReporterMiddlerware := middleware.NewNegroniBkStatsMid(app.DisplayName)
		n.Use(apiReporterMiddlerware)

		apiReporterMiddlerware.Start()
	}

	n.UseHandler(handler)

	return n
}
