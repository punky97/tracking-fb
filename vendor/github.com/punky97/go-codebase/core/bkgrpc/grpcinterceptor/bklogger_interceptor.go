package grpcinterceptor

import (
	"context"
	"github.com/punky97/go-codebase/core/bkgrpc"
	"github.com/punky97/go-codebase/core/bkgrpc/common"
	"github.com/punky97/go-codebase/core/logger"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// AlertFilter --
type AlertFilter struct {
	Method    string
	Threshold time.Duration
}

// BkLoggerServerInterceptor --
func BkLoggerServerInterceptor(alertFilters []AlertFilter, defaultThres time.Duration, debugMode bool) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		_, funcName := common.SplitMethodName(info.FullMethod)
		start := time.Now()
		p := &peer.Peer{}

		if pInfo, ok := peer.FromContext(ctx); ok {
			p = pInfo
		}
		if debugMode {
			logger.BkLog.Infof("Start handle req %v from %v, body: %v", funcName, p.Addr, req)
		}

		resp, err := handler(ctx, req)

		d := time.Since(start)
		if debugMode {
			if err == nil || status.Convert(err).Code() == codes.NotFound || status.Convert(err).Message() == bkgrpc.ErrNotFoundData.Error() {
				logger.BkLog.Infof("Completed req %v (%v) in %v from %v", funcName, req, d, p.Addr)
			} else {
				var code = status.Convert(err).Code()
				if code == codes.DeadlineExceeded || code == codes.Canceled {
					logger.BkLog.Warnf("Timeout/Cancelled handle req %v from %v (%v) in %v: %v", funcName, p.Addr, req, d, err)
				} else {
					logger.BkLog.Warnf("Unexpected error when handle req %v from %v (%v) in %v: %v", funcName, p.Addr, req, d, err)
				}
			}
		}

		var alertThreshold = defaultThres
		for _, alert := range alertFilters {
			if alert.Method == funcName {
				alertThreshold = alert.Threshold
				break
			}
		}

		if d > alertThreshold {
			logger.BkLog.Warnf("Warn: take too much time to process %v from %v (%v) in %v", funcName, p.Addr, req, d)
		}

		return resp, err
	}
}
