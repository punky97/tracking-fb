package app

import (
	"github.com/punky97/go-codebase/core/apimono"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/transport/transhttp"
	"github.com/spf13/viper"
	"net/http"
	"tracking-fb/cmd/api/tracking-api/app/handlers/tracking"
)

type server struct {
	//Producer *queue.Producer
	bkRedis  *bkredis.RedisClient
	producer *queue.Producer
}

var s = &server{}

func NewServer(apimonoApp *apimono.App) {
	apimonoApp.AddRoutes(s.InitTrackingRoutes(apimonoApp.HTTPBasePath))
}

func OnClose() {
	//s.Producer.Close()
}

func (s *server) InitTrackingRoutes(basePath string) transhttp.Routes {
	rawPath := viper.GetString("http_client.path")
	if len(rawPath) == 0 {
		rawPath = "https://graph.facebook.com/v11.0"
	}
	return transhttp.Routes{
		transhttp.Route{
			Name:     "",
			Method:   http.MethodPost,
			BasePath: basePath,
			Pattern:  "/track",
			Handler: &tracking.TrackingHandler{
				RawPath: rawPath,
			},
		},
	}
}
