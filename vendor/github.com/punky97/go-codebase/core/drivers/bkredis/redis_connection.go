package bkredis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/punky97/go-codebase/core/config"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/reporter/constant"
	"github.com/punky97/go-codebase/core/reporter/notifier"
	"github.com/spf13/viper"
	"time"
)

const (
	DefaultTickerInterval       = 6 * 1e3 // 6 seconds in milis
	DefaultSlowQueriesThreshold = 10      // 10 query
	DefaultWarnThreshold        = 50      // 50 milis
)

// IsInSlot -
func IsInSlot(key string, slot redis.ClusterSlot) bool {
	s := Slot(key)
	return slot.Start <= s && s <= slot.End
}

// GetSlotID -
func GetSlotID(key string, slots []redis.ClusterSlot) string {
	s := Slot(key)
	for k := range slots {
		slot := slots[k]
		if slot.Start <= s && s <= slot.End {
			return fmt.Sprintf("%v-%v", slot.Start, slot.End)
		}
	}
	return ""
}

type RedisInterface interface {
	redis.UniversalClient
	GetClient() redis.UniversalClient
	GetClusterSlots() ([]redis.ClusterSlot, error)
	GetRedisSlot(key string) int
	GetRedisSlotID(key string) string
	GetWithCtx(ctx context.Context) redis.UniversalClient
}

// RedisClient --
type RedisClient struct {
	redis.UniversalClient
	Slots       []redis.ClusterSlot
	IsRedisSlow bool
}

type DetectSlowRedisQueryConf struct {
	Enabled              bool
	SlowQueriesCount     int64
	SlowQueriesThreshold int64
	TickerInterval       int64
	WarnThreshold        int64
	SlackChannelUrl      string
	EnableLog            bool
}

// NewConnection -- open connection to db
func NewConnection(ctx context.Context, conn Connection) (*RedisClient, error) {
	var err error

	c, err := conn.BuildClient()

	if err != nil {
		logger.BkLog.Error("Could not build redis client, details: ", err)
		return nil, err
	}

	pong, err := c.Ping(ctx).Result()
	if err != nil {
		logger.BkLog.Error("Could not ping to redis, details: ", err)
		return nil, err
	}
	logger.BkLog.Info("Ping to redis: ", pong)
	cs := getClusterInfo(ctx, c)

	return getBkRedis(ctx, c, cs, viper.GetBool("elasticsearch_apm.enable")), nil
}

func getBkRedis(ctx context.Context, c redis.UniversalClient, cs []redis.ClusterSlot, apmTracing bool) *RedisClient {
	rd := &RedisClient{c, cs, false}
	conf := getSlowQueriesDetectConf()
	logger.BkLog.Info("Start redis check slow query ", conf)

	if conf.Enabled {
		slackNotifier := notifier.NewSlackNotifier()

		go func() {
			ticker := time.NewTicker(time.Duration(conf.TickerInterval) * time.Millisecond)
			for t := range ticker.C {
				start := time.Now()
				warnThreshold := time.Duration(conf.WarnThreshold) * time.Millisecond
				_, err := c.Ping(ctx).Result()
				elapsed := time.Since(start)

				if conf.EnableLog {
					payload, _ := json.Marshal(conf)
					logger.BkLog.Infof("Isslow: %v .Check redis took %v. conf: %v", rd.IsRedisSlow, elapsed, string(payload))
				}

				if err != nil {
					if !rd.IsRedisSlow {
						logger.BkLog.Error("Could not ping to redis, details: ", err)
						slackNotifier.Notify(slackNotifier.Format(constant.ReportLog{
							ReportType: constant.ServiceInPressure,
							Priority:   constant.ReportAlert,
							Data: map[string]interface{}{
								"app":    "redis",
								"desc":   "redis slow",
								"source": config.GetHostName(),
								"detail": fmt.Sprintf("redis slow: %v", t),
							},
						}))
					}
					rd.IsRedisSlow = true
					continue
				}

				if elapsed > warnThreshold {
					conf.SlowQueriesCount++
				} else {
					if rd.IsRedisSlow {
						logger.BkLog.Infof("Redis back to normal")
						slackNotifier.Notify(slackNotifier.Format(constant.ReportLog{
							ReportType: constant.ServiceInPressure,
							Priority:   constant.ReportResolve,
							Data: map[string]interface{}{
								"app":    "redis",
								"desc":   "redis back to normal",
								"source": config.GetHostName(),
								"detail": fmt.Sprintf("Redis :ngg: :onn: %v", t),
							},
						}))
					}
					rd.IsRedisSlow = false
					conf.SlowQueriesCount = 0
				}

				if !rd.IsRedisSlow && conf.SlowQueriesCount >= conf.SlowQueriesThreshold {
					// slow query or redis down
					rd.IsRedisSlow = true
					logger.BkLog.Info("[RD_DBG] Redis slow ", t)
					slackNotifier.Notify(slackNotifier.Format(constant.ReportLog{
						ReportType: constant.ServiceInPressure,
						Priority:   constant.ReportAlert,
						Data: map[string]interface{}{
							"app":    "redis",
							"desc":   "redis in pressure",
							"source": config.GetHostName(),
							"detail": fmt.Sprintf("Redis slow or down: %v", t),
						},
					}))
				}
			}
		}()
	}
	return rd
}
func getSlowQueriesDetectConf() *DetectSlowRedisQueryConf {
	conf := &DetectSlowRedisQueryConf{
		Enabled:              viper.GetBool("redis.enable_slow_detect"),
		SlowQueriesThreshold: viper.GetInt64("redis.slow_query_count_threshold"), // number of slow query for alert
		TickerInterval:       viper.GetInt64("redis.ticker_interval"),
		WarnThreshold:        viper.GetInt64("redis.slow_query_time_threshold"), // threshold for detect a query is slow or not
		SlackChannelUrl:      viper.GetString("redis.slow_alert_slack_channel"),
		EnableLog:            viper.GetBool("redis.enable_slow_detect_log"),
	}

	if conf.TickerInterval == 0 {
		conf.TickerInterval = DefaultTickerInterval
	}

	if conf.WarnThreshold == 0 {
		conf.WarnThreshold = DefaultWarnThreshold
	}

	if conf.SlowQueriesThreshold == 0 {
		conf.SlowQueriesThreshold = DefaultSlowQueriesThreshold
	}
	return conf
}

func getClusterInfo(ctx context.Context, c redis.UniversalClient) []redis.ClusterSlot {
	var cs = make([]redis.ClusterSlot, 0)
	if ci := c.ClusterInfo(ctx); ci.Err() == nil {
		csr := c.ClusterSlots(ctx)
		var err error
		cs, err = csr.Result()
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Cannot get cluster slots: %v", err))
		}
	}
	return cs
}

// NewConnectionFromExistedClient --
func NewConnectionFromExistedClient(ctx context.Context, c redis.UniversalClient) *RedisClient {
	cs := getClusterInfo(ctx, c)
	return &RedisClient{c, cs, false}
}

// Close -- close connection
func (br *RedisClient) Close() error {
	if br != nil {
		return br.UniversalClient.Close()
	}

	return nil
}

// GetClient --
func (br *RedisClient) GetClient() redis.UniversalClient {
	return br.UniversalClient
}

// GetClusterSlots -
func (br *RedisClient) GetClusterSlots(ctx context.Context) ([]redis.ClusterSlot, error) {
	res := br.ClusterSlots(ctx)
	return res.Result()
}

// GetRedisSlot -
func (br *RedisClient) GetRedisSlot(key string) int {
	return Slot(key)
}

// GetRedisSlotID -
func (br *RedisClient) GetRedisSlotID(key string) string {
	return GetSlotID(key, br.Slots)
}

// GetWithCtx --
func (br *RedisClient) GetWithCtx(ctx context.Context) redis.UniversalClient {
	return br.UniversalClient
}
