package app

import (
	"github.com/punky97/go-codebase/core/apimono"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/drivers/queue"
	"github.com/punky97/go-codebase/core/transport/transhttp"
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
	s.producer = apimonoApp.CreateRabbitMQProducerConnection(nil)
	apimonoApp.AddRoutes(s.InitTrackingRoutes(apimonoApp.HTTPBasePath))
}

func OnClose() {
	//s.Producer.Close()
}

func (s *server) InitTrackingRoutes(basePath string) transhttp.Routes {
	return transhttp.Routes{
		transhttp.Route{
			Name:     "",
			Method:   http.MethodPost,
			BasePath: basePath,
			Pattern:  "/track",
			Handler: &tracking.TrackingHandler{
				Producer: s.producer,
			},
		},
	}
}
