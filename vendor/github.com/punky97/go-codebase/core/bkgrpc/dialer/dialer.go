package dialer

import (
	"github.com/punky97/go-codebase/core/bkgrpc/balancer"
	"github.com/punky97/go-codebase/core/bkgrpc/grpcresolver"
	"github.com/punky97/go-codebase/core/bkgrpc/timeout"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"time"
)

// DialOption allows optional config for dialer
type DialOption func(name string) (grpc.DialOption, error)

// WithBackoff adding backoff config
func WithBackoff() DialOption {
	return func(name string) (grpc.DialOption, error) {
		return grpc.WithBackoffConfig(grpc.BackoffConfig{MaxDelay: 1.0 * time.Second}), nil
	}
}

// WithTimeout add timeout context
func WithTimeout(source string, t time.Duration, retry uint) grpc.UnaryClientInterceptor {
	return timeout.UnaryClientInterceptor(source, timeout.WithTimeout(t), timeout.WithRetry(retry))
}

// WithBalancer enables client side load balancing
func WithBalancer(cc *consul.Client) DialOption {
	return WithLocalBalancer(cc, false)
}

// WithLocalBalancer enables client side load balancing
func WithLocalBalancer(cc *consul.Client, localPreferred bool) DialOption {
	return func(name string) (grpc.DialOption, error) {
		lb := balancer.RoundRobin(grpcresolver.ForConsul(cc), localPreferred)
		// TODO: use the new balancer APIs in balancer package
		return grpc.WithBalancer(lb), nil
	}
}

// Dial returns a load balanced grpc client conn with tracing interceptor
func Dial(startOnReady bool, name string, opts ...DialOption) (*grpc.ClientConn, error) {
	dialopts := []grpc.DialOption{
		grpc.WithInsecure(),
	}

	for _, fn := range opts {
		opt, err := fn(name)
		if err != nil {
			return nil, fmt.Errorf("config error: %v", err)
		}
		dialopts = append(dialopts, opt)
	}

	if startOnReady {
		dialopts = append(dialopts, grpc.WithBlock())
	}

	if viper.GetInt("dms.max_recv_size") > 0 {
		maxSize := viper.GetInt("dms.max_recv_size")
		dialopts = append(dialopts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*maxSize)))
	}

	conn, err := grpc.Dial(name, dialopts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", name, err)
	}

	return conn, nil
}
