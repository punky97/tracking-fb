package debug

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/punky97/go-codebase/core/utils/compress"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

const _maxWaitTime int64 = 60

var DefaultProfilingPort = 8877

type AppProfilingHandler struct {
	profileSetting *ProfilingSetting
}

func NewAppProfilingHandler(profileSetting *ProfilingSetting) *AppProfilingHandler {
	return &AppProfilingHandler{
		profileSetting: profileSetting,
	}
}

func (h *AppProfilingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actionReq := r.URL.Query().Get("req")

	if actionReq == "start" {
		if h.profileSetting.IsStarted() {
			transhttp.RespondError(w, http.StatusBadRequest, "Profiling is started")
			return
		}

		h.profileSetting.Start()
		transhttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "profiling",
		})
		return
	} else if actionReq == "stop" {
		if !h.profileSetting.IsStarted() {
			transhttp.RespondError(w, http.StatusBadRequest, "Profiling is not running")
			return
		}

		tmpFile, err := h.StopProfiling()
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error when stop profiling: %v", err))
			transhttp.RespondError(w, http.StatusInternalServerError, err.Error())
		} else {
			defer os.Remove(tmpFile)
			transhttp.RespondFile(w, r, tmpFile)
		}
		return
	} else if actionReq == "start_and_wait" {
		waitTime, _ := strconv.ParseInt(r.URL.Query().Get("wait_time"), 10, 64)
		if waitTime <= 0 || waitTime > _maxWaitTime {
			transhttp.RespondError(w, http.StatusBadRequest, fmt.Sprintf("wait_time must be in range of 1 to %v", _maxWaitTime))
			return
		}

		if !h.profileSetting.IsStarted() {
			h.profileSetting.Start()
			time.Sleep(time.Duration(waitTime) * time.Second)
		}

		tmpFile, err := h.StopProfiling()
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error when stop profiling: %v", err))
			transhttp.RespondError(w, http.StatusInternalServerError, err.Error())
		} else {
			defer os.Remove(tmpFile)
			transhttp.RespondFile(w, r, tmpFile)
		}
		return
	} else if actionReq == "status" {
		if h.profileSetting.IsStarted() {
			transhttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"status": "profiling",
			})
		} else {
			transhttp.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"status": "idle",
			})
		}

		return
	}

	transhttp.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
		"error": fmt.Sprintf("Unknown req: %v", actionReq),
	})
}

func (h *AppProfilingHandler) StopProfiling() (string, error) {
	files := h.profileSetting.Stop()
	tmpFile, err := ioutil.TempFile("", "pprof.tar.gz")
	defer tmpFile.Close()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when create temp dir: %v", err))
		return "", errors.New("cannot create file")
	}
	if err := compress.GzipTarCompress(files, tmpFile); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when compress file: %v", err))
		return "", errors.New("cannot compress file")
	} else {
		logger.BkLog.Infof("Tmp file: %v", tmpFile.Name())
		return tmpFile.Name(), nil
	}
}
