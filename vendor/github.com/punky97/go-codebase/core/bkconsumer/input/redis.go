package input

import (
	"context"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/logger"
	"runtime"
	"strings"
	"time"
)

const (

	// RedisMode --
	RedisMode = "redis"
)

// RedisInputConf --
type RedisInputConf struct {
	Type string   `json:"type"`
	Keys []string `json:"keys"`
}

// KV --
type KV struct {
	Key   string
	Value string
}

// RedisInput --
type RedisInput struct {
	ctx context.Context
	*bkredis.RedisClient
	Out <-chan KV
}

// BuildRedisInput --
func BuildRedisInput(ctx context.Context, inp *RedisInputConf) *RedisInput {
	redis, _ := bkredis.NewConnection(ctx, bkredis.DefaultRedisConnectionFromConfig())
	ch := make(chan KV)

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.BkLog.Info("Closing redis input")
				redis.Close()
				close(ch)
				return
			default:
				r := redis.BLPop(ctx, 1*time.Second, inp.Keys...)
				s, err := r.Result()
				if err != nil {
					continue
				}
				if len(s) < 2 {
					logger.BkLog.Warn("Error input from redis: ", strings.Join(s, ","))
					continue
				}
				ch <- KV{s[0], s[1]}
			}
		}
	}()
	return &RedisInput{ctx: ctx, RedisClient: redis, Out: ch}
}

// Handle --
func (ri *RedisInput) Handle(ch <-chan KV, fn func(<-chan KV), thread int) {
	if thread < 1 {
		thread = runtime.NumCPU()
	}
	for i := 0; i < thread; i++ {
		go fn(ch)
	}
	for {
		select {
		case <-ri.ctx.Done():
			logger.BkLog.Info("Stop handling request")
			return
		}
	}
}
