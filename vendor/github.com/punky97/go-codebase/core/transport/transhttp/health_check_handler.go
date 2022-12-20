package transhttp

import "net/http"

type HealthCheckHandler struct {
}

func NewHealthCheckHandler() *HealthCheckHandler {
	return &HealthCheckHandler{}
}

func (h *HealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
