package bkconsumer

import (
	"context"
	"flag"
	"fmt"
	"github.com/punky97/go-codebase/core/bkconsumer/reporter"
	"github.com/punky97/go-codebase/core/bkmicro"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/debug"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils"
	cron2 "github.com/robfig/cron"
	"net/http"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"reflect"
)

const (
	DefaultTaskPreparationTime = 20
	DefaultTimeExpire          = 1 * time.Minute
)

// BkTask --
type BkTask interface {
	// GetHandlers returns handler(s) with replica option for each one
	GetHandlers() map[string]interface{}
	// GetDmsNames returns dms names that handler(s) want to interact to
	GetDmsNames() []string
	// SetDmsClient worker will returns client connections after initializing.
	// this function is for handle saving dms client
	SetDmsClient(map[string]*grpc.ClientConn)
	// SetProducer same with SetDmsClient but for producer (currently just for rabbitmq)
	SetProducer(*queue.Producer)
	// Init initialize handler. it will be called by worker internally
	Init(*Worker)
	// Close also be called by worker when closing
	Close(*Worker)
}

// TaskInterface will be deprecated soon. Please use BkTask instead
type TaskInterface interface {
	// GetHandlers returns handler(s) with replica option for each one
	GetHandlers() map[string]HandlerWOption
	// GetDmsNames returns dms names that handler(s) want to interact to
	GetDmsNames() []string
	// SetDmsClient worker will returns client connections after initializing.
	// this function is for handle saving dms client
	SetDmsClient(map[string]*grpc.ClientConn)
	// SetProducer same with SetDmsClient but for producer (currently just for rabbitmq)
	SetProducer(*queue.Producer)
	// Init initialize handler. it will be called by worker internally
	Init(*Worker)
	// Close also be called by worker when closing
	Close(*Worker)
}

// Bootstrapper --
var Bootstrapper = &Bootstrap{F: make([]TaskInterface, 0)}

// Bootstrap --
type Bootstrap struct {
	F []TaskInterface
	S []BkTask
}

// AddConsumerTask --
func (b *Bootstrap) AddConsumerTask(t TaskInterface) {
	b.F = append(b.F, t)
}

// AddScheduleTask --
func (b *Bootstrap) AddScheduleTask(t BkTask) {
	b.S = append(b.S, t)
}

// Worker --
type Worker struct {
	bkmicro.App
	// Common
	// task definition files
	TaskDefinitionURI string
	task              *Task
	scheduler         *Scheduler
	stopped           bool
	name              string
}

// Callback --
type Callback func(*Worker)

// NewWorker --
func NewWorker(onClose Callback) *Worker {
	w := &Worker{App: bkmicro.App{Name: "worker"}}

	w.initDefaultFlags()
	w.initCommon()

	// handle sigterm
	utils.HandleSigterm(func() {
		logger.BkLog.Infof("Stopping...")
		if onClose != nil {
			onClose(w)
		}
		w.Stop()
		debug.StopProfiling()
		notifier.Close()
	})
	return w
}

func (w *Worker) initCommon() {
	w.InitLogger()
	w.InitConfig()
	w.InitSdConnection()
	w.InitDebug()
	w.InitAlert()
}

// InitDefaultFlags -- init default flags of bkmicro app
func (w *Worker) initDefaultFlags() {
	// flags for logger
	flag.StringVar(&w.LoggerType, "logger-type", "file", "Logger type: file or default")
	flag.BoolVar(&w.LoggerDev, "logger-dev", false, "is logger for dev?")

	// flags for config
	flag.StringVar(&w.ConfigType, "config-type", "file", "Configuration type: file or remote")
	flag.StringVar(&w.ConfigFile, "config-file", "", "Configuration file")
	flag.StringVar(&w.ConfigRemoteAddress, "config-remote-address", "", "Configuration remote address. ip:port")
	flag.StringVar(&w.ConfigRemoteKeys, "config-remote-keys", "", "Configuration remote keys. Seperate by ,")

	// flags for service discovery
	flag.BoolVar(&w.SdEnable, "sd-enable", true, "Enable register to service discovery or not")
	flag.StringVar(&w.SdAddress, "sd-address", "127.0.0.1:8500", "Service discovery server address. ip:port")

	// flags for Distributed Tracing System
	var unused string
	var unusedBool bool
	flag.BoolVar(&unusedBool, "tracing-enable", false, "Enable tracing or not")
	flag.StringVar(&unused, "tracing-address", "", "Tracing server address. ip:port")
	// allow loopback ip?
	flag.BoolVar(&w.AllowLoopbackIP, "allow-loopback", false, "Allow loopback ip for registering")
	// timeout
	flag.Int64Var(&w.DefaultDMSTimeout, "dms-timeout", 0, "DMS client timeout")

	// app name
	flag.StringVar(&w.DisplayName, "app-name", "", "App name")
	if len(w.DisplayName) == 0 {
		w.DisplayName = w.Name
	}

	// enable profiling at start or not
	flag.BoolVar(&w.EnableProfiling, "profiling", false, "Enable profiling from start or not")

	// flags for http
	flag.StringVar(&w.HTTPHost, "http-host", "", "HTTP listen host")
	flag.Int64Var(&w.HTTPPort, "http-port", int64(debug.DefaultProfilingPort), "HTTP listen port")

	flag.Parse()
}

// CtxKey --
type CtxKey string

func (w *Worker) loadTaskDef(defFile string) *Task {
	var taskDefinition string
	var err error
	logger.BkLog.Debug("Loading task....")
	if defFile != "" {
		taskDefinition, err = config.ReadRawFile(defFile)
	} else if w.ConfigLoaded {
		w.TaskDefinitionURI = viper.GetString("task.def")
		switch w.ConfigType {
		case bkmicro.ConfigTypeFile:
			taskDefinition, err = config.ReadRawFile(w.TaskDefinitionURI)
		case bkmicro.ConfigTypeRemote:
			var b []byte
			b, err = config.GetFromConsulKV(w.ConfigRemoteAddress, w.TaskDefinitionURI)
			taskDefinition = string(b)
		}
	}

	if err != nil || taskDefinition == "" {
		logger.BkLog.Fatal("Cannot read task definition", w.TaskDefinitionURI, err)
	}

	ctx := context.Background()
	taskCtx := context.WithValue(ctx, CtxKey("taskctx"), "task context")
	newCtx, cancelFunc := context.WithCancel(taskCtx)
	taskDefStruct := parseTaskDef(taskDefinition)

	var appName string
	if len(w.DisplayName) > 0 && w.DisplayName != "worker" {
		appName = w.DisplayName
	} else {
		if len(Bootstrapper.F) > 0 {
			appName = reflect.TypeOf(Bootstrapper.F[0]).Elem().PkgPath() + "." + reflect.TypeOf(Bootstrapper.F[0]).Elem().Name()
		}
	}

	timeShutDownExpire := DefaultTimeExpire
	expire := viper.GetInt64("admin.consumer.shut_down_expire")
	if expire > 0 {
		timeShutDownExpire = time.Duration(expire) * time.Minute
	}

	task := Task{
		TaskDefinition:  taskDefStruct,
		consumers:       make(map[string]ConsumerDef),
		consumerHandler: make(map[string]HandlerWOption),
		cancelFunc:      cancelFunc, ctx: newCtx,
		taskMetrics:        reporter.NewConsumerWorkerMetrics(appName),
		timeShutDownExpire: timeShutDownExpire,
	}

	w.task = &task
	logger.BkLog.Debug("Loading task....done")
	return &task
}

// Walk --
func (w *Worker) Walk() {
	logger.BkLog.Debug("Start consumer")
	w.task.start()
	w.Stop()
}

// PrepareTask --
func (w *Worker) PrepareTask(defFile string) *Task {
	w.loadTaskDef(defFile)

	w.task.prepare()

	return w.task
}

// BuildHandler --
func BuildHandler(h MsgHandler, replica int64) *HandlerWOption {
	ignoreMetrics := false
	if disableMetrics, ok := h.(DisableMetricTask); ok {
		ignoreMetrics = disableMetrics.IsDisableMetric()
	}

	return &HandlerWOption{MsgHandler: h, Replica: replica, ignoreMetrics: ignoreMetrics}
}

// PrepareHandlers --
func (w *Worker) PrepareHandlers(m map[string]HandlerWOption) {
	w.task.InitializeHandler(m)
}

// PrepareHandler --
func (w *Worker) PrepareHandler(h HandlerWOption) {
	m := make(map[string]HandlerWOption)
	m[DefaultConsumerName] = h
	w.task.InitializeHandler(m)
}

// GetDmsClient --
func (w *Worker) GetDmsClient(dms ...string) map[string]*grpc.ClientConn {
	return w.InitGRPCClients(dms...)
}

// GetProducer --
func (w *Worker) GetProducer() *queue.Producer {
	if w.task.producer != nil {
		return w.task.producer.Producer
	}
	return nil
}

// GetProducerFromConf --
func (w *Worker) GetProducerFromConf() *queue.Producer {
	p := queue.NewRMQProducerFromDefConf()
	if p == nil {
		logger.BkLog.Info("nil producer")
	} else {
		if err := p.Connect(); err != nil {
			logger.BkLog.Fatal("Cannot connect to rabbitmq. Please check configuration file for more information", err)
		}
		p.Start()
		utils.HandleSigterm(func() {
			if p != nil {
				logger.BkLog.Debug("Stop producer")
				p.Close()
			}
		})
	}
	return p
}

func (w *Worker) prepareScheduler() {
	ctx := context.Background()
	taskCtx := context.WithValue(ctx, CtxKey("schedulerctx"), "scheduler context")
	newCtx, cancelFunc := context.WithCancel(taskCtx)

	var appName string
	if len(w.DisplayName) > 0 && w.DisplayName != "worker" {
		appName = w.DisplayName
	} else {
		if len(Bootstrapper.S) > 0 {
			appName = reflect.TypeOf(Bootstrapper.S[0]).Elem().PkgPath() + "." + reflect.TypeOf(Bootstrapper.S[0]).Elem().Name()
		}
	}
	w.scheduler = NewScheduler(appName, newCtx, cancelFunc, []*ScheduleJobOption{})
}

// PrepareScheduleTask --
func (w *Worker) PrepareScheduleTask(m map[string]interface{}) {
	for k, v := range m {
		if t, ok := v.(ScheduleTask); ok {
			// get data from config
			retries := viper.GetInt64(fmt.Sprintf("schedulers.%v.retries", t.GetName()))
			cron := viper.GetString(fmt.Sprintf("schedulers.%v.cron", t.GetName()))
			boundTimeStr := viper.GetString(fmt.Sprintf("schedulers.%v.bound_time", t.GetName()))
			timeType := viper.GetString(fmt.Sprintf("schedulers.%v.type", t.GetName()))
			preparationTime := viper.GetInt(fmt.Sprintf("schedulers.%v.prepare_time", t.GetName()))

			if len(cron) == 0 && len(timeType) == 0 {
				logger.BkLog.Warnf("Cannot find right configuration for %v, conf: %v",
					t.GetName(), fmt.Sprintf("schedulers.%v.cron", t.GetName()))
				continue
			}
			var sjo ScheduleJobOption
			f := func(timeType, cron string) (c string, boundDate time.Time) {
				d := time.Time{}
				if len(cron) > 0 && timeType != OneTimeMode {
					c = cron
					// parseCron
					sch, err := cron2.Parse(cron)
					if err != nil {
						logger.BkLog.Fatalf("Cannot parse cron string %v\n", cron)
						return
					}
					d = sch.Next(time.Now()).Add(DefaultBoundDate * time.Minute)
				} else {
					now := time.Now()
					if preparationTime == 0 {
						preparationTime = DefaultTaskPreparationTime
					}
					future := now.Add(time.Duration(preparationTime) * time.Second)
					c = fmt.Sprintf("%v", future.Format("05 04 15 _2 Jan Mon"))
					future = future.Add(DefaultBoundDate * time.Minute)
					d = future
				}

				if len(boundTimeStr) > 0 {
					tempDate, err := time.Parse("2006-01-02 15:04:05 -0700", boundTimeStr)
					if err != nil {
						logger.BkLog.Warnf("Invalid bound date for job %v: %v\n", t.GetName(), boundTimeStr)
					} else {
						d = tempDate
					}
				}
				boundDate = d
				return
			}
			sjo.cron, sjo.boundDate = f(timeType, cron)
			sjo.timeType = timeType
			sjo.st = t
			sjo.retries = retries

			sjo.ignoreMetric = false
			if disableMetric, ok := v.(DisableMetricTask); ok {
				sjo.ignoreMetric = disableMetric.IsDisableMetric()
			}

			w.scheduler.AddTask(&sjo)
		} else {
			logger.BkLog.Warn("Cannot convert task %v to ScheduleTask", k)
		}
	}
}

// Schedule --
func (w *Worker) Schedule() {
	logger.BkLog.Info("Scheduling tasks")
	w.scheduler.Schedule(w.runApi)
	w.Stop()
}

// Stop stops worker
func (w *Worker) Stop() {
	if w.stopped {
		logger.BkLog.Info("Call Stop method multiple times")
		return
	}
	w.stopped = true
	if w.task != nil {
		logger.BkLog.Info("Stop consumer")
		defer w.task.stop()
	}
	if w.scheduler != nil {
		logger.BkLog.Info("Stop scheduler")
		w.scheduler.stop()
	}
	time.Sleep(2 * time.Second)
}

func (w *Worker) runApi() {
	hs := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", w.HTTPHost, w.HTTPPort),
		IdleTimeout:  70 * time.Second,
		ReadTimeout:  40 * time.Second,
		WriteTimeout: 70 * time.Second,
	}
	hs.Handler = transhttp.NewRouter(w.initRoutes(), "Scheduler", 1000, false)
	err := hs.ListenAndServe()
	if err == http.ErrServerClosed {
		logger.BkLog.Infof("Server closed")
	}
}

type PodScrapeInfo struct {
	Name       string                `json:"name"`
	Schedulers []*ScheduleScrapeInfo `json:"schedulers"`
}

func (w *Worker) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	res := PodScrapeInfo{}
	if len(w.name) > 0 {
		res.Name = w.name
	} else {
		nameInit := r.URL.Query().Get("name")
		if len(nameInit) > 0 {
			w.name = nameInit
		}
	}
	res.Schedulers = make([]*ScheduleScrapeInfo, 0)
	scrapeTime := time.Now().Unix()
	for _, job := range w.scheduler.ScheduleJob {
		job.ScheduleScrapeInfo.ScrapeTime = scrapeTime
		res.Schedulers = append(res.Schedulers, &job.ScheduleScrapeInfo)

	}
	transhttp.RespondJSON(rw, 200, res)
}

// InitRoutes -- Initialize our routes
func (w *Worker) initRoutes() transhttp.Routes {
	return transhttp.Routes{
		transhttp.Route{
			Name:     "Status of route",
			Method:   http.MethodGet,
			BasePath: "/v1",
			Pattern:  "/status",
			Handler:  w,
			Timeout:  1000,
		},
	}
}
