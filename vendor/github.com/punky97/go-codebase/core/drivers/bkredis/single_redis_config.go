package bkredis

import (
	"github.com/go-redis/redis/v8"
	"github.com/punky97/go-codebase/core/logger"
	"time"
)

// Connection -- redis connection
type SingleConnection struct {
	network      string
	address      string
	password     string
	db           int
	maxRetries   int
	poolSize     int
	readTimeout  int
	writeTimeout int
}

func (conn *SingleConnection) BuildClient() (redis.UniversalClient, error) {
	if conn.address == "" {
		return nil, ErrorMissingRedisAddress
	}

	if conn.readTimeout == 0 {
		conn.readTimeout = DefaultReadTimeout
	}

	if conn.writeTimeout == 0 {
		conn.writeTimeout = DefaultWriteTimeout
	}

	logger.BkLog.Debugf("redis single - address: %v, pass: %v, db: %v, pollSize: %v, readTimeout: %v ms, writeTimeout: %v ms",
		conn.address, conn.password, conn.db, conn.poolSize, conn.readTimeout, conn.writeTimeout)

	return redis.NewClient(
		&redis.Options{
			Addr:         conn.address,
			Password:     conn.password, // no password set
			DB:           conn.db,       // use default DB
			PoolSize:     conn.poolSize,
			PoolTimeout:  time.Second * 4,
			ReadTimeout:  time.Millisecond * time.Duration(conn.readTimeout),
			WriteTimeout: time.Millisecond * time.Duration(conn.writeTimeout),
		},
	), nil
}
