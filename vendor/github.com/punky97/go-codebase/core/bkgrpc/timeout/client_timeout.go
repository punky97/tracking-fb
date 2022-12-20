package timeout

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/punky97/go-codebase/core/bkgrpc/common"
	"github.com/punky97/go-codebase/core/bkgrpc/meta"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/punky97/go-codebase/core/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"time"
)

const _defaultSleepRetryUnit = 50 * time.Millisecond

// UnaryClientInterceptor returns a new retrying unary client interceptor.
//
// The default configuration of the interceptor is to not retry *at all*. This behaviour can be
// changed through options (e.g. WithMax) on creation of the interceptor or on call (through grpc.CallOptions).
func UnaryClientInterceptor(source string, optFuncs ...CallOption) grpc.UnaryClientInterceptor {
	intOpts := reuseOrNewWithCallOptions(defaultOptions, optFuncs)

	slackNoti := notifier.DefaultNotifier()
	slackCriticalNoti := notifier.CriticalNotifier()

	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		grpcOpts, retryOpts := filterCallOptions(opts)
		callOpts := reuseOrNewWithCallOptions(intOpts, retryOpts)
		// short circuit for simplicity, and avoiding allocations.
		var lastErr error
		var header, trailer metadata.MD
		for attempt := uint(0); attempt <= callOpts.maxAttempt; attempt++ {
			callCtx, optTimeout := perCallContext(parentCtx, callOpts)
			startTime := time.Now()
			grpcOpts = append(grpcOpts, grpc.Header(&header), grpc.Trailer(&trailer))
			lastErr = invoker(callCtx, method, req, reply, cc, grpcOpts...)

			if lastErr == nil {
				return nil
			}
			if isContextTimeout(lastErr) && (optTimeout != 0 && time.Since(startTime) > optTimeout) {
				dmsList := header.Get(meta.TimeOutDmsPodNameKey)
				dmsPodName := "Not Detected"
				if len(dmsList) > 0 {
					dmsPodName = dmsList[0]
					logger.BkLog.Infof("[ClientTimeOut Reporter] Completed handling request from: %v", dmsPodName)
				}
				serviceName, methodName := common.SplitMethodName(method)
				reportLog := &constant.ReportLog{
					ReportType: constant.DMSTimeoutReport,
					Priority:   constant.ReportAlert,
					Data: map[string]interface{}{
						"service": serviceName,
						"method":  methodName,
						"timeout": callOpts.perCallTimeout.Nanoseconds(),
						"req":     req,
						"source":  source,
						"dms":     dmsPodName,
					},
				}

				go func() {
					defer func() {
						if r := recover(); r != nil {
							logger.BkLog.Errorw(fmt.Sprintf("Panic when notify: %v", r))
						}
					}()

					// only notify in last timeout attempt
					if attempt >= callOpts.maxAttempt {
						slackNoti.Notify(slackNoti.Format(*reportLog))
					}
					logger.BkLog.Warnf("client timeout %v after %v(retries): %v", method, attempt, reportLog)
				}()
			} else if isContextCancel(lastErr) {
				serviceName, methodName := common.SplitMethodName(method)
				reportLog := &constant.ReportLog{
					ReportType: constant.DMSCancelReport,
					Data: map[string]interface{}{
						"service": serviceName,
						"method":  methodName,
						"req":     req,
						"source":  source,
					},
				}

				logger.BkLog.Warnf("req %v is cancelled, %v", method, reportLog)
			}

			if !isRetriable(lastErr, callOpts) {
				break
			} else {
				logger.BkLog.Warnf("Client reties %v %v times: %v", method, attempt, lastErr)
				time.Sleep(_defaultSleepRetryUnit * time.Duration(attempt))
			}
		}

		if lastErr != nil &&
			(status.Convert(lastErr).Code() == codes.Unimplemented || (status.Convert(lastErr).Code() == codes.Unavailable)) {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						logger.BkLog.Errorw(fmt.Sprintf("Panic when notify: %v", r))
					}
				}()

				serviceName, methodName := common.SplitMethodName(method)

				var desc string
				code := status.Convert(lastErr).Code()
				if code == codes.Unimplemented {
					desc = fmt.Sprintf("Method %v hasnt been implemented yet", methodName)
				} else {
					desc = fmt.Sprintf("Service %v is unavailable", serviceName)
				}

				report := map[string]interface{}{
					"app":    serviceName,
					"desc":   desc,
					"detail": lastErr.Error(),
					"extras": []map[string]string{},
					"source": source,
				}

				logger.BkLog.Errorw(fmt.Sprintf("Critical error on service %v: %v", serviceName, lastErr.Error()), "method", methodName)

				slackCriticalNoti.Notify(slackNoti.Format(constant.ReportLog{
					ReportType: constant.CriticalError,
					Priority:   constant.ReportAlert,
					Data:       report,
				}))
			}()
		}

		return lastErr
	}
}

func isRetriable(err error, opts *options) bool {
	if len(opts.codes) == 0 {
		return true
	}

	if stt, ok := status.FromError(err); ok {
		for _, c := range opts.codes {
			if stt.Code() == c {
				return true
			}
		}
	}

	return false
}

func isContextTimeout(err error) bool {
	return status.Code(err) == codes.DeadlineExceeded
}

func isContextCancel(err error) bool {
	return status.Code(err) == codes.Canceled
}

func perCallContext(parentCtx context.Context, callOpts *options) (context.Context, time.Duration) {
	ctx := parentCtx
	if callOpts.perCallTimeout != 0 {
		ctx, _ = context.WithTimeout(ctx, callOpts.perCallTimeout)
	}

	var mdClone metautils.NiceMD

	if callOpts.ignoreCache {
		mdClone = metautils.ExtractOutgoing(ctx).Clone().Set(meta.XIgnoreCacheKey, "1")
	}

	if len(mdClone) == 0 {
		mdClone = metautils.ExtractOutgoing(ctx).Clone()
	}
	if callOpts.idempotent != "" {
		mdClone = mdClone.Set(meta.XIdempotentKey, callOpts.idempotent)
	} else {
		mdClone = mdClone.Set(meta.XIdempotentKey, utils.GenerateRandomString2(10))
	}

	if len(mdClone) > 0 {
		ctx = mdClone.ToOutgoing(ctx)
	}

	return ctx, callOpts.perCallTimeout
}
