package tracking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	beekit_utils "github.com/punky97/go-codebase/core/utils"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"time"
	"tracking-fb/dto"
	facebook_tracking "tracking-fb/pkg/facebook-tracking"
	"tracking-fb/pkg/utils"
)

type TrackingHandler struct {
	Producer *queue.Producer
	RawPath  string
}

type TrackingResponse struct {
	Success      bool   `json:"success,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (h *TrackingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	action := &dto.TrackingBody{}
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

	start := time.Now()
	var extraMsg string
	logger.BkLog.Info("Start tracking event: %v", action.EventType)
	defer func() {
		logger.BkLog.Info("Complete process msg, took: %v, extra msg: %v", time.Since(start), extraMsg)
	}()

	pixels := h.getPixel()
	trackingBody := h.convertAction(r, action)
	if action == nil {
		extraMsg += " Drop due to can't convert action"
		transhttp.RespondJSON(w, http.StatusOK, TrackingResponse{
			Success:      false,
			ErrorMessage: "missing event type",
		})
	}

	for _, pixel := range pixels {
		utils.DoSomethingWithRetry(func() (interface{}, error) {
			err := h.sendEvent(r.Context(), pixel, trackingBody)
			if err != nil {
				logger.BkLog.Errorw("Could not send event", "error", err.Error(), "pid", pixel.Id)
			}
			return err == nil, err
		}, 5, "sen event tracking")
	}

	transhttp.RespondJSON(w, http.StatusOK, TrackingResponse{
		Success: true,
	})
}

func (h *TrackingHandler) getPixel() []*dto.Pixel {

	return facebook_tracking.GetPixelByCode("main_pixel")
}

func (h *TrackingHandler) convertAction(r *http.Request, body *dto.TrackingBody) map[string]interface{} {
	params := body.Payload
	params["event_name"] = body.EventType
	ipAddress := r.Header.Get("X-Real-IP")
	userAgent := r.Header.Get("User-Agent")

	userData := map[string]interface{}{
		"client_ip_address": ipAddress,
		"client_user_agent": userAgent,
	}

	hashFields := []string{
		"em", "ph", "ge", "db",
		"ln", "fn", "ct", "zp",
		"st",
		"country", "external_id",
	}
	originFields := []string{
		"client_ip_address", "client_user_agent",
		"fbc", "fbp",
		"subscription_id", "lead_id",
		"fb_login_id",
	}

	for _, field := range hashFields {
		hash := beekit_utils.GetSHA256Hash("")
		if val, ok := params[field]; ok && val != "" {
			hash = cast.ToString(val)
		}

		userData[field] = hash
	}

	for _, field := range originFields {
		userData[field] = cast.ToString(params[field])
	}

	// Delete user data
	for _, field := range hashFields {
		delete(params, field)
	}

	for _, field := range originFields {
		delete(params, field)
	}
	delete(params, "source_url")

	params["user_data"] = userData
	return params
}

func (h *TrackingHandler) sendEvent(ctx context.Context, pixel *dto.Pixel, trackingPayload map[string]interface{}) error {
	logContext := logger.LoggerCtx(ctx)

	trackingPayload["event_id"] = fmt.Sprintf("%v_%v", pixel.Id, time.Now().Unix())

	body := make(map[string]interface{})
	body["data"] = []map[string]interface{}{trackingPayload}

	rawBody, err := json.Marshal(body)
	if err != nil {
		logContext.Errorw("Error marshal body data",
			"err", err.Error())
		return err
	}

	path := fmt.Sprintf("%v/%v/events?access_token=%v", h.RawPath, pixel.Id, pixel.Token)
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewBuffer(rawBody))
	if err != nil {
		logContext.Errorw("Error create request",
			"err", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100

	timeout := viper.GetInt("http_client.timeout")
	if timeout == 0 {
		timeout = 30
	}

	client := &http.Client{
		Timeout:   time.Duration(timeout) * time.Second,
		Transport: t,
	}

	res, err := client.Do(req)
	timeLog(start, "do http request")
	if err != nil {
		logContext.Errorw("Error send request",
			"err", err.Error())
		return err
	}

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		logContext.Errorw("Error read body",
			"extra_readable_info", err.Error())
		return nil
	}

	return nil
}

func timeLog(ts time.Time, task string) {
	elapsed := time.Since(ts)
	if elapsed < 200*time.Millisecond {
		return
	}
	logger.BkLog.Infof("%v took %v", task, elapsed)
}
