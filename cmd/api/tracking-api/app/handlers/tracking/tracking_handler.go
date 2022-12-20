package tracking

import (
	"encoding/json"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"net/http"
	"tracking-fb/dto"
	"tracking-fb/pkg/utils"
)

type TrackingHandler struct {
	Producer *queue.Producer
}

type TrackingResponse struct {
	Success      bool   `json:"success,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (h *TrackingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	action := &dto.Action{}
	if err := json.NewDecoder(r.Body).Decode(action); err != nil {
		logger.BkLog.Infof("Error during parse GetJobBodyRequest form %v", err)
		transhttp.RespondJSON(w, http.StatusOK, TrackingResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		})
	}

	if len(action.EventType) < 1 {
		transhttp.RespondJSON(w, http.StatusOK, TrackingResponse{
			Success:      false,
			ErrorMessage: "missing event type",
		})
	}
	// publish
	payload, _ := json.Marshal(action)
	utils.DoSomethingWithRetry(func() (interface{}, error) {
		err := h.Producer.PublishSimple("fb_tracking", payload)
		return err == nil, err
	}, 50, "publish to tracking")

	transhttp.RespondJSON(w, http.StatusOK, TrackingResponse{
		Success: true,
	})
}
