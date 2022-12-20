package bkredis

import (
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

type Connection interface {
	BuildClient() (redis.UniversalClient, error)
}

var (
	ErrorMissingRedisAddress = errors.New("missing redis address")
)

const (
	// Redis client type
	Sentinel = "sentinel"
	Cluster  = "cluster"

	// DefaultPoolSize --
	DefaultPoolSize     = 100
	DefaultReadTimeout  = 1000
	DefaultWriteTimeout = 1000
)

// DefaultRedisConnectionFromConfig -- load connection settings in config with default key
func DefaultRedisConnectionFromConfig() Connection {
	redisClientType := viper.GetString("redis.client_type")
	poolSize := viper.GetInt("redis.pool_size")
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}

	switch redisClientType {
	case Sentinel:
		return &SentinelConnection{
			masterGroup:       viper.GetString("redis_sentinel.sentinel_master"),
			sentinelAddresses: viper.GetStringSlice("redis_sentinel.sentinel_address"),
			password:          viper.GetString("redis.password"),
			db:                viper.GetInt("redis.db"),
			poolSize:          poolSize,
			readTimeout:       viper.GetInt("redis.read_timeout"),
			writeTimeout:      viper.GetInt("redis.write_timeout"),
		}
	case Cluster:
		return &ClusterConnection{
			clusterAddresses: viper.GetStringSlice("redis_cluster.cluster_address"),
			password:         viper.GetString("redis.password"),
			poolSize:         poolSize,
			readTimeout:      viper.GetInt("redis.read_timeout"),
			writeTimeout:     viper.GetInt("redis.write_timeout"),
		}
	default:
		return &SingleConnection{
			address:      viper.GetString("redis.address"),
			password:     viper.GetString("redis.password"),
			db:           viper.GetInt("redis.db"),
			readTimeout:  viper.GetInt("redis.read_timeout"),
			writeTimeout: viper.GetInt("redis.write_timeout"),
			poolSize:     poolSize,
		}
	}
}

func RedisBatchLayerConnection() Connection {
	redisClientType := viper.GetString("redis.batch.client_type")
	poolSize := viper.GetInt("redis.batch.pool_size")
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}

	switch redisClientType {
	case Sentinel:
		return &SentinelConnection{
			masterGroup:       viper.GetString("redis_sentinel.batch.sentinel_master"),
			sentinelAddresses: viper.GetStringSlice("redis_sentinel.batch.sentinel_address"),
			password:          viper.GetString("redis.batch.password"),
			db:                viper.GetInt("redis.batch.db"),
			poolSize:          poolSize,
			readTimeout:       viper.GetInt("redis.read_timeout"),
			writeTimeout:      viper.GetInt("redis.write_timeout"),
		}
	case Cluster:
		return &ClusterConnection{
			clusterAddresses: viper.GetStringSlice("redis_cluster.batch.cluster_address"),
			password:         viper.GetString("redis.batch.password"),
			poolSize:         poolSize,
			readTimeout:      viper.GetInt("redis.read_timeout"),
			writeTimeout:     viper.GetInt("redis.write_timeout"),
		}
	default:
		return &SingleConnection{
			address:      viper.GetString("redis.batch.address"),
			password:     viper.GetString("redis.batch.password"),
			db:           viper.GetInt("redis.batch.db"),
			poolSize:     poolSize,
			readTimeout:  viper.GetInt("redis.read_timeout"),
			writeTimeout: viper.GetInt("redis.write_timeout"),
		}
	}
}

func RedisServeLayerConnection() Connection {
	redisClientType := viper.GetString("redis.serve.client_type")
	poolSize := viper.GetInt("redis.serve.pool_size")
	if poolSize <= 0 {
		poolSize = DefaultPoolSize
	}

	switch redisClientType {
	case Sentinel:
		return &SentinelConnection{
			masterGroup:       viper.GetString("redis_sentinel.serve.sentinel_master"),
			sentinelAddresses: viper.GetStringSlice("redis_sentinel.serve.sentinel_address"),
			password:          viper.GetString("redis.serve.password"),
			db:                viper.GetInt("redis.serve.db"),
			poolSize:          poolSize,
			readTimeout:       viper.GetInt("redis.read_timeout"),
			writeTimeout:      viper.GetInt("redis.write_timeout"),
		}
	case Cluster:
		return &ClusterConnection{
			clusterAddresses: viper.GetStringSlice("redis_cluster.serve.cluster_address"),
			password:         viper.GetString("redis.serve.password"),
			poolSize:         poolSize,
			readTimeout:      viper.GetInt("redis.read_timeout"),
			writeTimeout:     viper.GetInt("redis.write_timeout"),
		}
	default:
		return &SingleConnection{
			address:      viper.GetString("redis.serve.address"),
			password:     viper.GetString("redis.serve.password"),
			db:           viper.GetInt("redis.serve.db"),
			poolSize:     poolSize,
			readTimeout:  viper.GetInt("redis.read_timeout"),
			writeTimeout: viper.GetInt("redis.write_timeout"),
		}
	}
}

// NewRedisConfig --
func NewRedisConfig(add string, db int) Connection {
	return &SingleConnection{
		address:      add,
		db:           db,
		poolSize:     DefaultPoolSize,
		readTimeout:  viper.GetInt("redis.read_timeout"),
		writeTimeout: viper.GetInt("redis.write_timeout"),
	}
}

// NewRedisConfigWithPool --
func NewRedisConfigWithPool(add string, db, poolSize int) Connection {
	return &SingleConnection{
		address:      add,
		db:           db,
		poolSize:     poolSize,
		readTimeout:  viper.GetInt("redis.read_timeout"),
		writeTimeout: viper.GetInt("redis.write_timeout"),
	}
}
