package meta

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/metadata"
	"strings"
)

func IsAllowCache(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	return metautils.ExtractIncoming(ctx).Get(XIgnoreCacheKey) != "1"
}

func GetIdempotencyKey(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return metautils.ExtractIncoming(ctx).Get(XIdempotentKey)
}

func GetTracingId(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return metautils.ExtractIncoming(ctx).Get(DMSTracingKey)
}

func GetDMSSourcePodName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return metautils.ExtractIncoming(ctx).Get(DMSSourcePodName)
}

func GetDMSMethodName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return metautils.ExtractIncoming(ctx).Get(DMSMethodName)
}

func GetDMSSourceIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	return metautils.ExtractIncoming(ctx).Get(DMSSourceIP)
}

func GetCtxData(ctx context.Context) map[string]string {
	metadata.FromOutgoingContext(ctx)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}

	rawDataSlice := md.Get(XCtxData)
	if len(rawDataSlice) > 0 {
		data := make(map[string]string)
		for _, d := range rawDataSlice {
			ds := strings.SplitN(d, "=", 2)
			if len(ds) != 2 {
				continue
			}

			data[ds[0]] = ds[1]
		}

		return data
	} else {
		return nil
	}
}
