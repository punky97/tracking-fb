package mongo

import (
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/punky97/go-codebase/core/logger"
	"time"
)

// NewMongoConnection -- open connection
func NewConnection(conn *Connection) (*BkMongo, error) {
	dialInfo := &mgo.DialInfo{
		Addrs:       conn.addresses,
		Database:    conn.databaseName,
		FailFast:    conn.failFast,
		Username:    conn.user,
		Password:    conn.password,
		Service:     conn.service,
		ServiceHost: conn.serviceHost,
		Mechanism:   conn.mechanism,
		PoolLimit:   conn.poolLimit,
		Direct:      conn.direct,
		Timeout: time.Second * func() time.Duration {
			if conn.timeout <= 0 {
				return 5
			}
			return conn.timeout
		}(),
	}

	session, err := mgo.DialWithInfo(dialInfo)

	if err != nil {
		logger.BkLog.Errorw(
			fmt.Sprintf("Could not create connection: %v", err),
			"connection", fmt.Sprintf("mongodb %v/%v with timeout %v seconds", conn.addresses, conn.databaseName, conn.timeout))
		return nil, err
	}
	err = session.Ping()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Could not ping to mongodb database, details: %v", err))
		return nil, err
	}

	session.SetCursorTimeout(0)
	return &BkMongo{session}, nil
}
