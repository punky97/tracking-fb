package bkredis

import (
	"github.com/go-redis/redis/v8"
	"github.com/punky97/go-codebase/core/logger"
	"time"
)

// Sentinel Connection -- redis connection
type SentinelConnection struct {
	masterGroup       string
	sentinelAddresses []string
	password          string
	db                int
	poolSize          int
	readTimeout       int
	writeTimeout      int
}

func (conn *SentinelConnection) BuildClient() (redis.UniversalClient, error) {
	if len(conn.sentinelAddresses) == 0 {
		return nil, ErrorMissingRedisAddress
	}

	masterGroup := conn.masterGroup
	if masterGroup == "" {
		masterGroup = "master"
	}

	if conn.readTimeout == 0 {
		conn.readTimeout = DefaultReadTimeout
	}

	if conn.writeTimeout == 0 {
		conn.writeTimeout = DefaultWriteTimeout
	}

	logger.BkLog.Debugf("redis sentinel - address: %v, pass: %v, db: %v, pollSize: %v, readTimeout: %v ms, writeTimeout: %v ms",
		conn.sentinelAddresses, conn.password, conn.db, conn.poolSize, conn.readTimeout, conn.writeTimeout)

	redisdb := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    masterGroup,
		SentinelAddrs: conn.sentinelAddresses,
		Password:      conn.password,
		DB:            conn.db,
		PoolSize:      conn.poolSize,
		PoolTimeout:   time.Second * 4,
		ReadTimeout:   time.Millisecond * time.Duration(conn.readTimeout),
		WriteTimeout:  time.Millisecond * time.Duration(conn.writeTimeout),
	})

	return redisdb, nil
}
