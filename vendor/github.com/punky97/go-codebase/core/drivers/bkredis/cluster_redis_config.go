package bkredis

import (
	"github.com/punky97/go-codebase/core/logger"
	"github.com/go-redis/redis/v8"
	"time"
)

// Cluster Connection -- redis connection
type ClusterConnection struct {
	clusterAddresses []string
	password         string
	poolSize         int
	readTimeout      int
	writeTimeout     int
}

func (conn *ClusterConnection) BuildClient() (redis.UniversalClient, error) {
	if len(conn.clusterAddresses) == 0 {
		return nil, ErrorMissingRedisAddress
	}
	logger.BkLog.Infof("Create cluster client to %v", conn.clusterAddresses)
	if conn.readTimeout == 0 {
		conn.readTimeout = DefaultReadTimeout
	}

	if conn.writeTimeout == 0 {
		conn.writeTimeout = DefaultWriteTimeout
	}

	logger.BkLog.Debugf("redis cluster - address: %v, pass: %v, pollSize: %v, readTimeout: %v ms, writeTimeout: %v ms",
		conn.clusterAddresses, conn.password, conn.poolSize, conn.readTimeout, conn.writeTimeout)

	redisdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        conn.clusterAddresses,
		Password:     conn.password,
		PoolSize:     conn.poolSize,
		PoolTimeout:  time.Second * 4,
		ReadTimeout:  time.Millisecond * time.Duration(conn.readTimeout),
		WriteTimeout: time.Millisecond * time.Duration(conn.writeTimeout),
	})

	return redisdb, nil
}
