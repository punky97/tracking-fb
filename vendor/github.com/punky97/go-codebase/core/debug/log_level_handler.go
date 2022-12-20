package debug

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
)

type LogLevelHandler struct {
}

func NewLogLevelHandler() *LogLevelHandler {
	return &LogLevelHandler{}
}

func (h *LogLevelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	var lvl zapcore.Level

	switch level {
	case "debug":
		lvl = zap.DebugLevel
	case "info":
		lvl = zap.InfoLevel
	case "warn":
		lvl = zap.WarnLevel
	case "error":
		lvl = zap.ErrorLevel
	case "fatal":
		lvl = zap.FatalLevel
	default:
		transhttp.RespondJSON(w, http.StatusBadRequest, map[string]string{
			"errors": "unknown log level",
		})
		return
	}

	logger.BkLog.Infof("Change level from %v to %v", logger.BkLog.GetLevel(), lvl)
	logger.BkLog.SetLevel(lvl)

	transhttp.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "success",
	})
	return
}
