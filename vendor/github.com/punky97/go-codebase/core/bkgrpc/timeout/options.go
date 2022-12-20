package timeout

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	defaultOptions = &options{
		perCallTimeout: 0, // disabled
		maxAttempt:     0, // not attempt
		codes:          []codes.Code{codes.DeadlineExceeded, codes.Canceled, codes.Unavailable},
	}
)

func WithTimeout(timeout time.Duration) CallOption {
	return CallOption{applyFunc: func(o *options) {
		o.perCallTimeout = timeout
	}}
}

func WithRetry(t uint) CallOption {
	return CallOption{applyFunc: func(o *options) {
		o.maxAttempt = t
	}}
}

func WithCodes(cs []codes.Code) CallOption {
	return CallOption{applyFunc: func(o *options) {
		o.codes = cs
	}}
}

func WithCache(cache bool) CallOption {
	return CallOption{applyFunc: func(o *options) {
		o.ignoreCache = !cache
	}}
}

func WithIdempotent(idempotent string) CallOption {
	return CallOption{applyFunc: func(o *options) {
		o.idempotent = idempotent
	}}
}

type options struct {
	perCallTimeout time.Duration
	// retry
	codes      []codes.Code
	maxAttempt uint
	idempotent string
	// cache
	ignoreCache bool
}

// CallOption is a grpc.CallOption that is local to grpc_retry.
type CallOption struct {
	grpc.EmptyCallOption // make sure we implement private after() and before() fields so we don't panic.
	applyFunc            func(opt *options)
}

func reuseOrNewWithCallOptions(opt *options, callOptions []CallOption) *options {
	if len(callOptions) == 0 {
		return opt
	}
	optCopy := &options{}
	*optCopy = *opt
	for _, f := range callOptions {
		f.applyFunc(optCopy)
	}
	return optCopy
}

func filterCallOptions(callOptions []grpc.CallOption) (grpcOptions []grpc.CallOption, retryOptions []CallOption) {
	for _, opt := range callOptions {
		if co, ok := opt.(CallOption); ok {
			retryOptions = append(retryOptions, co)
		} else {
			grpcOptions = append(grpcOptions, opt)
		}
	}
	return grpcOptions, retryOptions
}
