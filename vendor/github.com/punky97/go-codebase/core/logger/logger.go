package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/natefinch/lumberjack"
	"github.com/punky97/go-codebase/core/config"
	"io/ioutil"
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BkLog is logger
var BkLog *BkLogger
var hooks map[string]func(zapcore.Entry, map[string]interface{}) error

func init() {
	InitLoggerDefaultDev()
}

// InitLoggerDefault -- init logger default
func InitLoggerDefault() {
	// init production encoder config
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	// init production config
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig = encoderCfg
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stdout"}
	// build logger
	logger, _ := cfg.Build()
	logger = logger.WithOptions(
		zap.AddCallerSkip(1),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return registerCtxHooks(core, hookDistribution)
		}),
	)

	sugarLog := logger.Sugar()
	cfgParams := make(map[string]interface{})
	BkLog = &BkLogger{"", cfgParams, cfg.Level, logger, sugarLog}
}

// InitLoggerDefaultDev -- init logger dev
func InitLoggerDefaultDev() {
	// init production encoder config
	encoderCfg := zap.NewDevelopmentEncoderConfig()
	// init production config
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig = encoderCfg
	cfg.OutputPaths = []string{"stdout"}
	// build logger
	logger, _ := cfg.Build()
	logger = logger.WithOptions(
		zap.AddCallerSkip(1),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return registerCtxHooks(core, hookDistribution)
		}),
	)

	sugarLog := logger.Sugar()
	cfgParams := make(map[string]interface{})
	BkLog = &BkLogger{"", cfgParams, cfg.Level, logger, sugarLog}
}

// InitLoggerFile -- init logger write to file
func InitLoggerFile() {
	if _, err := os.Stat("./conf/log.toml"); os.IsNotExist(err) {
		viper.Set("output_path", "./log/app.log")
		viper.Set("max_size_in_mb", 10)
		viper.Set("max_backups", 10)
		viper.Set("max_age", 30)
	} else {
		config.ReadBkConfig("log", "./conf")

	}
	var err error
	BkLog, err = NewLogger(
		viper.GetString("output_path"),
		viper.GetInt("max_size_in_mb"),
		viper.GetInt("max_backups"),
		viper.GetInt("max_age"),
	)

	if err != nil {
		panic(fmt.Sprintf("Cannot create logger with the following error: %s", err))
	}
}

// InitLoggerFileDev -- init logger write to file with development config
func InitLoggerFileDev() {
	if _, err := os.Stat("./conf/log.toml"); os.IsNotExist(err) {
		viper.Set("output_path", "./log/app.log")
		viper.Set("max_size_in_mb", 10)
		viper.Set("max_backups", 10)
		viper.Set("max_age", 30)
	} else {
		config.ReadBkConfig("log", "./conf")

	}
	var err error
	BkLog, err = NewLoggerFileDev(
		viper.GetString("output_path"),
		viper.GetInt("max_size_in_mb"),
		viper.GetInt("max_backups"),
		viper.GetInt("max_age"),
	)

	if err != nil {
		panic(fmt.Sprintf("Cannot create logger with the following error: %s", err))
	}
}

// BkLogger is a utility struct for logging data in an extremely high performance system.
// We can use both Logger and SugarLog for logging. For more information,
// just visit https://godoc.org/go.uber.org/zap
type BkLogger struct {
	// tracing id
	TracingId string
	// configuration
	config   map[string]interface{}
	logLevel zap.AtomicLevel
	// Logger for logging
	Logger *zap.Logger
	// Sugar for logging
	*zap.SugaredLogger
}

func createWithConfig(cfg zap.Config) (*BkLogger, error) {
	logger, err := cfg.Build()
	if err != nil {
		fmt.Println(
			fmt.Sprintf("Cannot create logger from configuration. Please take a look at the config file again. %s - %s",
				cfg.OutputPaths[0], cfg.ErrorOutputPaths[0]))
		return nil, err
	}

	sugarLog := logger.Sugar()

	cfgParams := make(map[string]interface{})
	cfgParams["output"] = cfg.OutputPaths[0]
	cfgParams["errOutput"] = cfg.ErrorOutputPaths[0]
	return &BkLogger{"", cfgParams, cfg.Level, logger, sugarLog}, nil
}

func LoggerCtx(r context.Context) *BkLogger {
	l, ok := r.Value("bk_logger").(*BkLogger)
	if ok {
		return l
	}

	return BkLog
}

// NewLogger create new logger based on file path
func NewLogger(outFilePath string, maxSizeInMB, maxBackups, maxAge int) (*BkLogger, error) {

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   outFilePath,
		MaxSize:    maxSizeInMB, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   true,
		LocalTime:  true,
	})

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	atom := zap.NewAtomicLevelAt(zap.InfoLevel)
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		w,
		atom,
	)

	zapcore.NewTee()
	logger := zap.New(core,
		zap.AddCallerSkip(1),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return registerCtxHooks(core, hookDistribution)
		}))

	sugarLog := logger.Sugar()

	cfgParams := make(map[string]interface{})
	cfgParams["output"] = outFilePath
	cfgParams["maxSize"] = maxSizeInMB
	cfgParams["maxBackup"] = maxBackups
	cfgParams["maxAge"] = maxAge

	return &BkLogger{"", cfgParams, atom, logger, sugarLog}, nil
}

// NewLoggerFileDev create new logger based on file path
func NewLoggerFileDev(outFilePath string, maxSizeInMB, maxBackups, maxAge int) (*BkLogger, error) {

	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   outFilePath,
		MaxSize:    maxSizeInMB, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   true,
		LocalTime:  true,
	})

	cfg := zap.NewDevelopmentEncoderConfig()
	atom := zap.NewAtomicLevelAt(zap.InfoLevel)
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		w,
		atom,
	)

	logger := zap.New(core,
		zap.AddCallerSkip(1),
		zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return registerCtxHooks(core, hookDistribution)
		}))

	sugarLog := logger.Sugar()

	cfgParams := make(map[string]interface{})
	cfgParams["output"] = outFilePath
	cfgParams["maxSize"] = maxSizeInMB
	cfgParams["maxBackup"] = maxBackups
	cfgParams["maxAge"] = maxAge

	return &BkLogger{"", cfgParams, atom, logger, sugarLog}, nil
}

// NewLoggerWithCfgFile create logger from config file
//
// Config should be in this format
//
//	  {
//		  "level": "debug",
//		  "encoding": "json",
//		  "outputPaths": ["path"],
//		  "errorOutputPaths": ["stderr", "path"],
//		  "encoderConfig": {
//		  	"messageKey": "message",
//			  "levelKey": "level",
//			  "levelEncoder": "lowercase"
//		  }
//	  }
func NewLoggerWithCfgFile(cfgFile string) (*BkLogger, error) {
	file, err := os.Open(cfgFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rawJSON, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		fmt.Println("Cannot validate log configuration file")
		return nil, err
	}
	return createWithConfig(cfg)
}

// Use this func to register a hook to Zap
func RegisterHook(appName string, hookCallback func(zapcore.Entry, string)) {
	if hooks == nil {
		hooks = make(map[string]func(zapcore.Entry, map[string]interface{}) error)
	}

	hooks[appName] = func(entry zapcore.Entry, _ map[string]interface{}) error {
		hookCallback(entry, appName)
		return nil
	}
}

// Use this func to register a context hook to Zap (both entry and fields)
func RegisterContextHook(appName string, hookCallback func(zapcore.Entry, map[string]interface{}, string)) {
	if hooks == nil {
		hooks = make(map[string]func(zapcore.Entry, map[string]interface{}) error)
	}

	hooks[appName] = func(entry zapcore.Entry, fields map[string]interface{}) error {
		hookCallback(entry, fields, appName)
		return nil
	}
}

func hookDistribution(entry zapcore.Entry, fields map[string]interface{}) error {
	if len(hooks) > 0 {
		for _, hookCallback := range hooks {
			hookCallback(entry, fields)
		}
	}

	return nil
}

// Close will flush log to file
func (bkl *BkLogger) Close() {
	bkl.Logger.Sync()
}

func AddLogCtx(ctx context.Context, bkLogger BkLogger, tracingId string) context.Context {
	ctxLogger := &bkLogger
	ctxLogger.TracingId = tracingId
	ctx = context.WithValue(ctx, "bk_logger", ctxLogger)

	return ctx
}
