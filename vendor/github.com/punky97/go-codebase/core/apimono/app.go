package apimono

import (
	"context"
	"flag"
	"fmt"
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/debug"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/registry"
	team_distribution "github.com/punky97/go-codebase/core/reporter/team-distribution"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils"
	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
	"github.com/spf13/viper"
	"github.com/urfave/negroni"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type BkAppType string

const (
	// BkAppTypeAPI -- api app
	BkAppTypeAPI BkAppType = "api"
	// BkAppTypeDMS -- dms app
	BkAppTypeDMS BkAppType = "dms"
	// BkAppTypeAPIGW -- api gateway app
	BkAppTypeAPIGW BkAppType = "apigw"
	// BkAppTypeDMSGW -- dms gateway app
	BkAppTypeDMSGW BkAppType = "dmsgw"
	// BkAppTypeConsumer -- consumer app
	BkAppTypeConsumer BkAppType = "consumer"

	BatchLayer string = "batch"
	ServeLayer string = "serve"

	DefaultApiCacheDuration = 60
	ApiCacheEnable          = "enable"
	ApiCacheDisable         = "disable"
	DefaultTracingHeader    = "x-ray-id"
)

var (
	// DefaultGatewayNamespace -- namespace for gateway service
	DefaultGatewayNamespace = "go.dpmx.gateway"
	// DefaultAPINamespace -- namespace for api service (restfull api)
	DefaultAPINamespace = "go.dpmx.api"
	// DefaultGRPCNamespace -- namespace for grpc service (dms)
	DefaultGRPCNamespace = "go.dpmx.grpc"

	// ConfigTypeFile -- config type FILE
	ConfigTypeFile = "file"
	// ConfigTypeRemote -- config type REMOTE
	ConfigTypeRemote = "remote"
	// SupportedConfigTypes -- list config type supported
	SupportedConfigTypes = []string{ConfigTypeFile, ConfigTypeRemote}

	// LoggerTypeFile -- logger type FILE, log will be write to file
	LoggerTypeFile = "file"
	// LoggerTypeDefault -- logger type DEFAULT, log will be write to stdout
	LoggerTypeDefault = "default"
	// SupportedLoggerTypes -- list logger type supported
	SupportedLoggerTypes = []string{LoggerTypeFile, LoggerTypeDefault}
)

type ProfilingSetting struct {
	sync.RWMutex
	closed    chan struct{}
	start     chan struct{}
	stop      chan struct{}
	isStart   bool
	limitTime int64
}

func (p *ProfilingSetting) IsStarted() bool {
	p.RLock()
	defer p.RUnlock()
	return p.isStart
}

func (p *ProfilingSetting) SetStart(started bool) {
	p.Lock()
	defer p.Unlock()
	p.isStart = started
}

// App -- beekit micro app
type App struct {
	// Common
	Name        string
	DisplayName string
	Description string
	Version     string
	Type        BkAppType
	// Configuration
	ConfigLoaded        bool
	ConfigType          string
	ConfigFile          string
	ConfigRemoteAddress string
	ConfigRemoteKeys    string
	// Logger
	LoggerType string
	LoggerDev  bool
	// Service discovery
	SdEnable  bool
	SdAddress string
	RegClient registry.Registry
	RegSvc    *registry.Service
	// allow to register loopback
	AllowLoopbackIP bool
	ServiceAddr     string
	// HTTP
	HTTPHost     string
	HTTPPort     int64
	HTTPBasePath string
	Middlewares  []negroni.Handler
	Routes       transhttp.Routes
	// gGRPC
	GrpcHost string
	GrpcPort int64

	// Middleware toggle
	DisableLogMiddleware bool
	// Timeout
	DefaultAPITimeout int64
	DefaultDMSTimeout int64
	// Silent on dev
	SilentOnDev bool
	// cleanup
	CleanupAtStart bool
	// enable profiling at start
	EnableProfiling bool

	//apply dirty field interceptor
	RemoveDirtyFieldEnable bool

	EnableLimitSelectFieldAndRecord bool
	//apply limit field select from db
	DefaultFieldSelect bool
	//apply limit record select from db
	DefaultLimitEnable bool
	// Should OS exit
	Exit chan bool
	// To indicate that bkmicroApp is running
	IsRunning bool

	consulCheckInterval int64

	MetaData map[string]string

	ApiCache          string
	ApiCacheDuration  int
	HttpTracingHeader string
	DmsTracingHeader  string
	IstioInjection    bool
}

type MockApp struct {
}

// InitDefaultFlags -- init default flags of bkmicro app
func (app *App) InitDefaultFlags() {
	// flags for logger
	flag.StringVar(&app.LoggerType, "logger-type", "file", "Logger type: file or default")
	flag.BoolVar(&app.LoggerDev, "logger-dev", false, "is logger for dev?")

	// flags for config
	flag.StringVar(&app.ConfigType, "config-type", "file", "Configuration type: file or remote")
	flag.StringVar(&app.ConfigFile, "config-file", "", "Configuration file")
	flag.StringVar(&app.ConfigRemoteAddress, "config-remote-address", "", "Configuration remote address. ip:port")
	flag.StringVar(&app.ConfigRemoteKeys, "config-remote-keys", "", "Configuration remote keys. Seperate by ,")

	// flags for service discovery
	flag.BoolVar(&app.SdEnable, "sd-enable", true, "Enable register to service discovery or not")
	flag.StringVar(&app.SdAddress, "sd-address", "127.0.0.1:8500", "Service discovery server address. ip:port")

	// flags for http
	flag.StringVar(&app.HTTPHost, "http-host", "", "HTTP listen host")
	flag.Int64Var(&app.HTTPPort, "http-port", 8888, "HTTP listen port")

	// flags for grpc
	flag.StringVar(&app.GrpcHost, "grpc-host", "", "grpc listen host")
	flag.Int64Var(&app.GrpcPort, "grpc-port", 8899, "grpc listen port")

	// flags for Distributed Tracing System
	var unused string
	var unusedBool bool
	flag.BoolVar(&unusedBool, "tracing-enable", false, "Enable tracing or not")
	flag.StringVar(&unused, "tracing-address", "", "Tracing server address. ip:port")

	// allow loopback ip?
	flag.BoolVar(&app.AllowLoopbackIP, "allow-loopback", false, "Allow loopback ip for registering")
	flag.StringVar(&app.ServiceAddr, "service-addr", "", "Service address for registering to consul")

	// timeout
	flag.Int64Var(&app.DefaultAPITimeout, "api-timeout", 0, "Timeout of API in ms")
	flag.Int64Var(&app.DefaultDMSTimeout, "dms-timeout", 0, "Timeout of DMS in ms")

	// silent on dev
	flag.BoolVar(&app.SilentOnDev, "silent", false, "Silent metric log")

	// cleanup
	flag.BoolVar(&app.CleanupAtStart, "cleanup-at-start", false, "For cleaning up all the mess at start or not")

	// app name
	flag.StringVar(&app.DisplayName, "app-name", "", "App name")
	if len(app.DisplayName) == 0 {
		app.DisplayName = app.Name
	}

	// enable profiling at start or not
	flag.BoolVar(&app.EnableProfiling, "profiling", false, "Enable profiling from start or not")

	flag.BoolVar(&app.RemoveDirtyFieldEnable, "remove-dirty-field-enable", false, "Enable remove dirty fields or not")

	flag.BoolVar(&app.EnableLimitSelectFieldAndRecord, "enable-limit-select-field-and-record", false, "Enable limit select field and record")
	flag.BoolVar(&app.DefaultFieldSelect, "default-field-select", false, "Enable default field select")
	flag.BoolVar(&app.DefaultLimitEnable, "default-limit-enable", false, "Enable default limit enable")
	flag.Int64Var(&app.consulCheckInterval, "consul-check-interval", 30, "Enable default interval consul health check")
	flag.BoolVar(&app.IstioInjection, "istio-inject", false, "Enable istio injection")
}

// ParseFlags -- parse flags
func (app *App) ParseFlags() {
	flag.Parse()
}

// Info -- show info of app
func (app *App) Info() string {
	return fmt.Sprintf("App type: %v. App name: %v. App version %v",
		app.Type, app.Name, app.Version)
}

func (app *App) AddMetadata(k string, v string) {
	if app.MetaData == nil {
		app.MetaData = make(map[string]string)
	}

	app.MetaData[k] = v
}

func (app *App) GetMetadata(k string) string {
	v, _ := app.MetaData[k]
	return v
}

// AddRoutes -- add HTTP routes
func (app *App) AddRoutes(routes transhttp.Routes, mrs ...negroni.Handler) {
	// This use only for local environment
	if viper.GetBool("development.disable_route_auth") {
		for i := range routes {
			routes[i].AuthInfo.Enable = false
			routes[i].Timeout = -1
		}
	}

	for _, route := range routes {
		if route.ApiCacheDuration <= 0 {
			if app.ApiCacheDuration <= 0 {
				route.ApiCacheDuration = DefaultApiCacheDuration
			} else {
				route.ApiCacheDuration = app.ApiCacheDuration
			}
		}

		if app.ApiCache == ApiCacheEnable {
			route.ApiCache = true
		} else if app.ApiCache == ApiCacheDisable {
			route.ApiCache = false
		}

		app.Routes = append(app.Routes, route)
	}

	app.Middlewares = mrs
}

// PreCheck -- pre check some conditional
func (app *App) PreCheck() {
	if utils.IsStringEmpty(app.Name) {
		log.Panic("must set bkmicro app name")
	}
	if utils.IsStringEmpty(string(app.Type)) {
		log.Panic("must set bkmicro app type")
	}
	if app.Type == BkAppTypeAPI {
		if utils.IsStringEmpty(app.HTTPBasePath) {
			log.Panic("must set http base path for bkmicro API app")
		}
	}
	if app.IstioInjection {
		// wait istio-proxy init
		time.Sleep(10 * time.Second)
	}
}

// InitLogger -- initializes logger
func (app *App) InitLogger() {
	loggerType := utils.StringTrimSpace(strings.ToLower(app.LoggerType))
	// if loggerType is empty, set default to `file` type`
	if utils.IsStringEmpty(loggerType) {
		loggerType = LoggerTypeFile
	}
	// check supported loggerType
	if !utils.IsStringSliceContains(SupportedLoggerTypes, loggerType) {
		log.Panicf("Not supported logger type: %v", loggerType)
	}

	switch loggerType {
	case LoggerTypeFile:
		if app.LoggerDev {
			logger.InitLoggerFileDev()
		} else {
			logger.InitLoggerFile()
		}

	case LoggerTypeDefault:
		if app.LoggerDev {
			logger.InitLoggerDefaultDev()
		} else {
			logger.InitLoggerDefault()
		}
	}

	app.HttpTracingHeader = viper.GetString("http.tracing.header")
	if app.HttpTracingHeader == "" {
		app.HttpTracingHeader = DefaultTracingHeader
	}

	app.DmsTracingHeader = viper.GetString("dms.tracing.key")
	if app.DmsTracingHeader == "" {
		app.DmsTracingHeader = meta.DMSTracingKey
	}

	logger.BkLog.Info("Logger loaded")
}

// InitConfig -- initializes config
func (app *App) InitConfig() {
	configType := utils.StringTrimSpace(strings.ToLower(app.ConfigType))
	configFile := utils.StringTrimSpace(app.ConfigFile)
	configRemoteAddress := utils.StringTrimSpace(app.ConfigRemoteAddress)
	configRemoteKeys := utils.StringSlice(
		utils.StringTrimSpace(app.ConfigRemoteKeys), ",",
	)

	// if configType is empty, set default to `file` type
	if utils.IsStringEmpty(configType) {
		configType = ConfigTypeFile
	}

	// check supported configType
	if !utils.IsStringSliceContains(SupportedConfigTypes, configType) {
		logger.BkLog.Panicf("Not support config type: %v", configType)
	}

	result := false

	switch configType {
	case ConfigTypeFile:
		if utils.IsStringEmpty(configFile) {
			// read by default
			result = config.ReadBkConfig("conf", "./conf", ".")
		} else {
			// read by input file
			result = config.ReadBkConfigByFile(configFile)
		}

	case ConfigTypeRemote:
		if utils.IsStringEmpty(configRemoteAddress) {
			logger.BkLog.Panic("Remote config server address is empty")
		}
		if len(configRemoteKeys) < 1 {
			logger.BkLog.Warn("This app has no config defined")
		} else {
			for index := 0; index < len(configRemoteKeys); index++ {
				isMerge := true
				if index == 0 {
					isMerge = false
				}
				remoteKey := utils.StringTrimSpace(configRemoteKeys[index])
				if utils.IsStringEmpty(remoteKey) {
					continue
				}
				if remoteKey[0:1] != "/" {
					logger.BkLog.Errorw(fmt.Sprintf("Invalid key: %v. Remote key must start with /", remoteKey))
					continue
				}
				valueBytes, err := config.GetFromConsulKV(app.ConfigRemoteAddress, remoteKey)
				if err != nil {
					logger.BkLog.Warnf("Could not get key %v from consul. Details: %v", remoteKey, err)
					continue
				}
				err = config.LoadConfig("toml", valueBytes, isMerge)
				if err != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Could not load config from remote key %v. Details: %v", remoteKey, err))
					continue
				}
				logger.BkLog.Infof("Loaded config remote key %v", remoteKey)
			}
		}

		result = true

	}

	if !result {
		logger.BkLog.Panic("Could not load config")
	}

	app.ConfigLoaded = true
	logger.BkLog.Info("Config loaded")
}

// GetCookieSecretKey -- get cookie secret key
func (app *App) GetCookieSecretKey() (keyBytes []byte) {
	if !app.ConfigLoaded {
		logger.BkLog.Panic("Config must init first")
	}

	configType := utils.StringTrimSpace(strings.ToLower(app.ConfigType))
	configRemoteAddress := utils.StringTrimSpace(app.ConfigRemoteAddress)

	var err error

	switch configType {
	case ConfigTypeFile:
		path := viper.GetString("cookies.secret_key_path")
		keyBytes, err = ioutil.ReadFile(path)
		if err != nil {
			logger.BkLog.Panicf("Error while reading cookie secret key at file %v", path)
		}
	case ConfigTypeRemote:
		if utils.IsStringEmpty(configRemoteAddress) {
			logger.BkLog.Panic("Remote config server address is empty")
		}
		keyBytes, err = config.GetFromConsulKV(app.ConfigRemoteAddress, "/cookie/secret.key")
		if err != nil {
			logger.BkLog.Panicf("Error get cookie secret key from remote config server. Details: %v", err)
		}
	}

	return
}

// InitJWT -- initializes jwt
func (app *App) InitJWT() {
	if !app.ConfigLoaded {
		logger.BkLog.Panic("Config must init first")
	}

	configType := utils.StringTrimSpace(strings.ToLower(app.ConfigType))
	configRemoteAddress := utils.StringTrimSpace(app.ConfigRemoteAddress)

	var err error
	var signBytes, verifyBytes, encryptKeyBytes []byte
	switch configType {
	case ConfigTypeFile:
		signBytes, err = ioutil.ReadFile(viper.GetString("jwt.private_key_path"))
		if err != nil {
			logger.BkLog.Panicf("Error get private key from file. Details: %v", err)
		}

		verifyBytes, err = ioutil.ReadFile(viper.GetString("jwt.public_key_path"))
		if err != nil {
			logger.BkLog.Panicf("Error get public key from file. Details: %v", err)
		}

		encryptKeyBytes, err = ioutil.ReadFile(viper.GetString("jwt.encrypt_key_path"))
		if err != nil {
			logger.BkLog.Panicf("Error get encrypt key from file. Details: %v", err)
		}

	case ConfigTypeRemote:
		if utils.IsStringEmpty(configRemoteAddress) {
			logger.BkLog.Panic("Remote config server address is empty")
		}
		signBytes, err = config.GetFromConsulKV(app.ConfigRemoteAddress, "/jwt/private.key")
		if err != nil {
			logger.BkLog.Panicf("Error get private key from remote config server. Details: %v", err)
		}

		verifyBytes, err = config.GetFromConsulKV(app.ConfigRemoteAddress, "/jwt/public.key")
		if err != nil {
			logger.BkLog.Panicf("Error get public key from remote config server. Details: %v", err)
		}

		encryptKeyBytes, err = config.GetFromConsulKV(app.ConfigRemoteAddress, "/jwt/encrypt.key")
		if err != nil {
			logger.BkLog.Panicf("Error get encrypt key from remote config server. Details: %v", err)
		}

	}

	err = utils.InitJWT(signBytes, verifyBytes, encryptKeyBytes)
	if err != nil {
		logger.BkLog.Panicf("Could not load jwt. Detail %v", err)
	}
	logger.BkLog.Info("JWT loaded")
}

func (app *App) InitDirtyFieldChecking() {
	app.RemoveDirtyFieldEnable = viper.GetBool("dirty_field.enable")
	logger.BkLog.Infof("Inited model dirty field removing, status: %v", app.RemoveDirtyFieldEnable)
}

func (app *App) InitLimitRecordAndFieldSelect() {
	app.EnableLimitSelectFieldAndRecord = viper.GetBool("limit_field_select_and_record.enable")
	app.DefaultLimitEnable = viper.GetBool("limit_field_select_and_record.default_record")
	app.DefaultFieldSelect = viper.GetBool("limit_field_select_and_record.default_field")
}

func (app *App) InitApiCache() {
	app.ApiCache = viper.GetString("api.cache.status")
	app.ApiCacheDuration = viper.GetInt("api.cache.duration")
}

// InitSdConnection -- initializes connection to service discovery (consul)
func (app *App) InitSdConnection() {
	if app.SdEnable {
		sdAddress := utils.StringTrimSpace(app.SdAddress)
		ignoreLoopback := viper.GetBool("registry.ignore_loopback")
		logger.BkLog.Infof("App %v ignore_loopback is %v", app.Name, ignoreLoopback)
		if utils.IsStringNotEmpty(sdAddress) {
			if app.consulCheckInterval <= 0 {
				app.consulCheckInterval = viper.GetInt64("consul.consul_check_interval")
			}
			if app.consulCheckInterval <= 0 {
				app.consulCheckInterval = 30
			}

			var ctx context.Context

			grpcHealthCheck := viper.GetBool("consul.grpc_health_check")
			httpHealthCheck := viper.GetBool("consul.http_health_check")
			tcpHealthCheck := viper.GetBool("consul.tcp_health_check")
			if grpcHealthCheck {
				logger.BkLog.Infof("Enable consul GRPC health check mode.")
				ctx = context.WithValue(context.Background(), "consul_grpc_check_interval", time.Duration(app.consulCheckInterval)*time.Second)
			} else if httpHealthCheck {
				logger.BkLog.Infof("Enable consul HTTP health check mode.")
				ctx = context.WithValue(context.Background(), "consul_http_check_interval", time.Duration(app.consulCheckInterval)*time.Second)
			} else if tcpHealthCheck {
				logger.BkLog.Infof("Enable consul TCP health check mode.")
				ctx = context.WithValue(context.Background(), "consul_tcp_check_interval", time.Duration(app.consulCheckInterval)*time.Second)
			}

			app.RegClient = registry.NewRegistry(
				func(opts *registry.Options) {
					opts.Addrs = []string{sdAddress}
					opts.IgnoreLoopback = ignoreLoopback
					if ctx != nil {
						opts.Context = ctx
					}
				},
			)
			logger.BkLog.Info("Initialize connection to service discovery")
		} else {
			app.SdEnable = false
		}
	}
	if !app.SdEnable {
		logger.BkLog.Info("App is disable service discovery")
	}
}

// IsSdConnectionCreated -- check connection to service discovery (consul) is created
func (app *App) IsSdConnectionCreated() bool {
	if !app.SdEnable {
		return false
	}
	if app.RegClient == nil {
		logger.BkLog.Error("Connection to service discovery register is not create")
		return false
	}
	return true
}

// GetSdAddress -- get service discovery address
func (app *App) GetSdAddress() (sa string) {
	return utils.StringTrimSpace(app.SdAddress)
}

// SdRegister -- service discovery register
func (app *App) SdRegister() {
	if !app.IsSdConnectionCreated() {
		return
	}
	switch app.Type {
	case BkAppTypeAPI:
		app.RegSvc = app.generateAPIRegistryService(app.Routes)
	case BkAppTypeDMS:
		app.RegSvc = app.generateGRPCRegistryService()
	case BkAppTypeAPIGW, BkAppTypeDMSGW:
		app.RegSvc = app.generateGatewayRegistryService()
	}
	// deregister if needed
	if app.CleanupAtStart {
		_ = app.RegClient.DeregisterMalformedService(app.RegSvc)
	}
	err := app.RegClient.Register(app.RegSvc)
	if err != nil {
		logger.BkLog.Error("Cannot register service ", err)
	} else {
		logger.BkLog.Infof("Register service successful, id %v", app.RegSvc.Nodes[0].Id)
	}
}

// SdDeregister -- service discovery register
func (app *App) SdDeregister() {
	if !app.IsSdConnectionCreated() {
		return
	}
	err := app.RegClient.Deregister(app.RegSvc)
	if err != nil {
		logger.BkLog.Error("Cannot deregister gRPC service ", err)
	} else {
		logger.BkLog.Infof("Deregister service successful, id %v", app.RegSvc.Nodes[0].Id)
	}
}

// generateGatewayRegistryService -- generate registry service object of beekit micro service
func (app *App) generateGatewayRegistryService() *registry.Service {
	serviceName := buildGatewayRegistryServiceName(DefaultGatewayNamespace, app.Name)
	serviceVersion := app.Version
	// check if gateway is api or dms
	var port int
	if app.Type == BkAppTypeAPIGW {
		port = int(app.HTTPPort)
	} else if app.Type == BkAppTypeDMSGW {
		port = int(app.GrpcPort)
	}
	registeringAddr := app.ServiceAddr
	if len(registeringAddr) == 0 {
		registeringAddr = utils.MyIP(app.AllowLoopbackIP)
	}

	nodes := []*registry.Node{
		{
			Id:      buildRegistryNodeID(app.Name),
			Address: registeringAddr,
			Port:    port,
		},
	}

	return &registry.Service{
		Name:    serviceName,
		Version: serviceVersion,
		Nodes:   nodes,
	}
}

// generateAPIRegistryService -- generate registry service object of beekit micro service
func (app *App) generateAPIRegistryService(routes transhttp.Routes) *registry.Service {
	serviceName := BuildRegistryServiceName(DefaultAPINamespace, app.HTTPBasePath)
	serviceVersion := app.Version
	registeringAddr := app.ServiceAddr
	if len(registeringAddr) == 0 {
		registeringAddr = utils.MyIP(app.AllowLoopbackIP)
	}
	nodes := []*registry.Node{
		{
			Id:      buildRegistryNodeID(app.Name),
			Address: registeringAddr,
			Port:    int(app.HTTPPort),
		},
	}

	dmsTimeout := app.DefaultDMSTimeout
	if dmsTimeout <= 0 {
		dmsTimeout = viper.GetInt64("dms.timeout")
	}
	if dmsTimeout <= 0 {
		if app.LoggerDev {
			dmsTimeout = DefaultDevDmsTimeout
		} else {
			dmsTimeout = DefaultDmsTimeout
		}
	}

	endpoints := make([]*registry.Endpoint, 0)

	for _, route := range routes {
		e := &registry.Endpoint{}
		e.Name = route.Name
		e.Request = &registry.Value{
			Name: route.BasePath + route.Pattern,
			Type: route.Method,
		}
		if route.AuthInfo.Enable {
			e.AuthInfo = &registry.AuthInfo{
				Enable:         route.AuthInfo.Enable,
				TokenType:      route.AuthInfo.TokenType,
				RequiredFields: route.AuthInfo.RequiredFields,
				RestrictScopes: route.AuthInfo.RestrictScopes,
			}
		}

		e.Timeout = route.Timeout
		if route.Timeout == 0 {
			timeout := app.DefaultAPITimeout
			if timeout <= 0 {
				timeout = viper.GetInt64("api.timeout")
			}
			if timeout <= 0 {
				timeout = transhttp.DefaultTimeout
			}

			e.Timeout = timeout
		}

		endpoints = append(endpoints, e)
	}

	return &registry.Service{
		Name:      serviceName,
		IDName:    app.DisplayName,
		Version:   serviceVersion,
		Metadata:  app.MetaData,
		Nodes:     nodes,
		Endpoints: endpoints,
	}
}

// generateGRPCRegistryService -- generate registry service object of beekit micro service
func (app *App) generateGRPCRegistryService() *registry.Service {
	serviceName := BuildGRPCRegistryServiceName(DefaultGRPCNamespace, app.Name)
	serviceVersion := app.Version
	registeringAddr := app.ServiceAddr
	if len(registeringAddr) == 0 {
		registeringAddr = utils.MyIP(app.AllowLoopbackIP)
	}
	nodes := []*registry.Node{
		{
			Id:      buildRegistryNodeID(app.Name),
			Address: registeringAddr,
			Port:    int(app.GrpcPort),
		},
	}

	return &registry.Service{
		Name:    serviceName,
		IDName:  app.DisplayName,
		Version: serviceVersion,
		Nodes:   nodes,
	}
}

func (app *App) InitDebug() {
	// init debug
	var debugEnable = viper.GetBool("admin.debug_mode.enable")
	var profilingEnable = viper.GetBool("admin.profiling.enable")

	if profilingEnable {
		debugEnable = true
		debug.EnableProfiling(app.DisplayName)
	}

	if debugEnable {
		profilingPort := viper.GetInt("admin.profiling.port")
		if profilingPort <= 0 {
			profilingPort = debug.DefaultProfilingPort
		}

		if viper.GetBool("admin.debug_mode.enable_debug_log") {
			logger.BkLog.SetLevel(zap.DebugLevel)
		}

		go func() {
			// handle panic
			defer func() {
				if p := recover(); p != nil {
					logger.BkLog.Errorw(fmt.Sprintf("Panic: %v", p))
				}
			}()

			n := negroni.New()
			router := mux.NewRouter().StrictSlash(true)
			router.Methods(http.MethodGet).Path("/bk-debug/profiling").Handler(debug.NewAppProfilingHandler(debug.DefaultProfileSetting))
			router.Methods(http.MethodGet).Path("/bk-debug/log-level").Handler(debug.NewLogLevelHandler())
			router.Methods(http.MethodGet).Path("/bk-debug/stat").Handler(debug.NewStatHandler())

			n.UseHandler(router)

			addr := fmt.Sprintf(":%v", profilingPort)
			logger.BkLog.Infof("Starting debug server: %v", addr)
			if err := http.ListenAndServe(addr, n); err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Unable to start debug listener: %v", err))
			}
		}()
	}
}

func (app *App) InitAlert() {
	// init core-zaphook handler
	hookWebhook := viper.GetString("reporter.notifier.hook.slack.webhook")
	if len(hookWebhook) > 0 {
		hookMembers := viper.GetStringSlice("reporter.notifier.hook.slack.members")
		logger.BkLog.Infof("Init zap hook webhook alert with member: %v", hookMembers)
		zapHandler := team_distribution.NewTeamDistributeHandler(hookWebhook, hookMembers)
		logger.RegisterContextHook(config.GetHostName(), zapHandler.Handle)
	}
}

func buildRegistryNodeID(name string) string {
	return fmt.Sprintf("%s-%s", name, uuid.NewUUID().String())
}

// BuildGRPCRegistryServiceName -- build gRPC registry service name
func BuildGRPCRegistryServiceName(namespace, grpcServiceName string) string {
	return fmt.Sprintf("%s.%s", namespace, grpcServiceName)
}

func BuildRegistryServiceName(namespace, basePath string) string {
	name := strings.Join(utils.StringSlice(basePath, "/"), ".")
	return fmt.Sprintf("%s.%s", namespace, name)
}

func buildGatewayRegistryServiceName(namespace, serviceName string) string {
	return fmt.Sprintf("%s.%s", namespace, serviceName)
}
