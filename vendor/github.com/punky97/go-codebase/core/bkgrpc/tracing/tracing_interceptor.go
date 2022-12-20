package tracing

import (
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/spf13/cast"
	"google.golang.org/grpc"
)

func UnaryClientInterceptor(dmsTracingHeader string) grpc.UnaryClientInterceptor {
	return func(parentCtx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx := parentCtx

		tracingID := cast.ToString(ctx.Value(dmsTracingHeader))
		if tracingID != "" {
			var mdClone metautils.NiceMD
			mdClone = metautils.ExtractOutgoing(ctx).Clone().Set(meta.DMSTracingKey, tracingID)

			ctx = mdClone.ToOutgoing(ctx)
		}

		lastErr := invoker(ctx, method, req, reply, cc, opts...)
		return lastErr
	}
}

func StreamClientInterceptor(dmsTracingHeader string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		tracingID := cast.ToString(ctx.Value(dmsTracingHeader))
		if tracingID != "" {
			var mdClone metautils.NiceMD
			mdClone = metautils.ExtractOutgoing(ctx).Clone().Set(meta.DMSTracingKey, tracingID)

			ctx = mdClone.ToOutgoing(ctx)
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
