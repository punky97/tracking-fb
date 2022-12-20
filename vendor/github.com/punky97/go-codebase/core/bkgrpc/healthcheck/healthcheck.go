package healthcheck

import (
	"github.com/punky97/go-codebase/core/logger"
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"time"
)

type healthServer struct {
	Service string
}

func RegisterHealthCheckServer(s *grpc.Server, svcName string) {
	healthpb.RegisterHealthServer(s, &healthServer{Service: svcName})
}

func (hs *healthServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	if req.Service == "" {
		// TODO: return SERVICE_UNKNOWN for this case
		logger.BkLog.Warnf("Health Check request is missing service name")
		//return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVICE_UNKNOWN}, nil
	} else if req.Service != hs.Service {
		logger.BkLog.Warnf("Wrong service name actual %v, expect %v", req.Service, hs.Service)
		return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVICE_UNKNOWN}, nil
	}
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (hs *healthServer) Watch(req *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	if req.Service == "" {
		// TODO: return SERVICE_UNKNOWN for this case
		logger.BkLog.Warnf("Health Check request is missing service name")
		//_ = stream.Send(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVICE_UNKNOWN})
		//return status.Error(codes.Canceled, "Stream has ended.")
	} else if req.Service != hs.Service {
		logger.BkLog.Warnf("Wrong service name actual %v, expect %v", req.Service, hs.Service)
		_ = stream.Send(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVICE_UNKNOWN})
		return status.Error(codes.Canceled, "Stream has ended.")
	}

	for {
		select {
		// Status updated. Sends the up-to-date status to the client.
		case <-time.After(time.Second * 5):
			err := stream.Send(&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING})
			if err != nil {
				return status.Error(codes.Canceled, "Stream has ended.")
			}
		// Context done. Removes the update channel from the updates map.
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream has ended.")
		}
	}
}
