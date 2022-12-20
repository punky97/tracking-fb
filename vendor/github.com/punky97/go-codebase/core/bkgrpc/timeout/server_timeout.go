package timeout

import (
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc"
)

const _defaultIdempotentTimeoutSecond = 10

func UnaryServerInterceptor(host string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		var mdClone metautils.NiceMD
		//idempotentKey := meta.GetIdempotencyKey(ctx)
		mdClone = metautils.ExtractIncoming(ctx).Set(meta.TimeOutDmsPodNameKey, host)
		ctx = mdClone.ToIncoming(ctx)

		//cacheVal, hit, err := cache.BkCache.GetCacheAdv(idempotentKey)
		//if (hit == cache.CacheHitLocal || hit == cache.CacheHitRemote) && err == nil {
		//	if cacheSerialize, ok := cacheVal.([]byte); ok {
		//		err = codec.DecodeProto(cacheSerialize, )
		//		respPointer := reflect.New(reflect.TypeOf(resp).Elem()).Interface()
		//	}
		//}

		resp, err = handler(ctx, req)

		//code := common.GrpcErrorConvert(err)
		//if common.IsErrorCode(code.String()) {
		//	if err == nil {
		//		if protoMess, ok := resp.(proto.Message); ok {
		//			cache.BkCache.SetCacheProtoTTL(idempotentKey, protoMess, _defaultIdempotentTimeoutSecond)
		//			cache.BkCache.SetCacheProtoTTL(idempotentKey+":type", protoMess, _defaultIdempotentTimeoutSecond)
		//		}
		//	} else {
		//		retryMess.err = err
		//	}
		//}

		return resp, err
	}
}
