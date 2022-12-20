package grpcinterceptor

import (
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"context"
	"google.golang.org/grpc"
)

// valueMapStream
type ctxStream struct {
	grpc.ServerStream
	ctx context.Context
}

func StreamWithContext(ss grpc.ServerStream, ctx context.Context) *ctxStream {
	return &ctxStream{
		ServerStream: ss,
		ctx:          ctx,
	}
}

func (c *ctxStream) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}

	return c.ServerStream.Context()
}

// BkContextServerInterceptor --
func BkContextServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(pCtx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx := utils.NewDataContext(pCtx)
		// passing client context data
		ctxData := meta.GetCtxData(pCtx)
		for k, v := range ctxData {
			utils.SetCtxData(ctx, k, v)
		}

		// tracing logger
		ctx = logger.AddLogCtx(ctx, *logger.BkLog, meta.GetTracingId(ctx))

		resp, err := handler(ctx, req)
		return resp, err
	}
}

// BkContextServerStreamInterceptor --
func BkContextServerStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, pSs grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := utils.NewDataContext(pSs.Context())
		// passing client context data
		ctxData := meta.GetCtxData(pSs.Context())
		for k, v := range ctxData {
			utils.SetCtxData(ctx, k, v)
		}

		ctx = logger.AddLogCtx(ctx, *logger.BkLog, meta.GetTracingId(ctx))

		ss := StreamWithContext(pSs, ctx)
		return handler(srv, ss)
	}
}
