package debug

import (
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"net/http"
)

type statHandler struct {
}

type PodStatResponse struct {
	Pod           string                 `json:"pod"`
	LogLevel      string                 `json:"log_level"`
	ProfilingStat string                 `json:"profiling_stat"`
	ReporterStat  string                 `json:"reporter_stat"`
	NewRelicStat  string                 `json:"new_relic_stat"`
	NotifierConf  map[string]interface{} `json:"notifier_conf"`
}

func NewStatHandler() *statHandler {
	return &statHandler{}
}

func (h *statHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := PodStatResponse{
		Pod:           config.GetHostName(),
		LogLevel:      logger.BkLog.GetLevel(),
		ProfilingStat: h.GetProfilingStat(),
		ReporterStat:  h.GetReporterStat(),
		NotifierConf:  h.GetNotifierConf(),
	}

	transhttp.RespondJSON(w, http.StatusOK, resp)
}

func (h *statHandler) GetProfilingStat() string {
	if DefaultProfileSetting.name == "" {
		return "disable"
	}

	if DefaultProfileSetting.IsStarted() {
		return "profiling"
	} else {
		return "idle"
	}
}

func (h *statHandler) GetReporterStat() string {
	if reporter.IsMetricsEnable() {
		if reporter.IsAllMetricsEnable() {
			return "enable all"
		} else {
			return "enable core"
		}
	}

	return "disable"
}

func (h *statHandler) GetNotifierConf() map[string]interface{} {
	return reporter.IsNotifyEnable()
}
