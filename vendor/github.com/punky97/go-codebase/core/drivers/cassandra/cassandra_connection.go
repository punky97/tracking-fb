package cassandra

import (
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"github.com/gocql/gocql"
)

// BkCassandra --
type BkCassandra struct {
	*gocql.Session
}

/**
 * Censor sensitive data related to db connection when dumping
 */

// NewConnection -- open connection to db
func NewConnection(conn *Connection) (*BkCassandra, error) {
	var err error
	cluster := gocql.NewCluster(conn.Cluster...)
	cluster.Keyspace = conn.Keyspace

	if conn.Timeout > 0 {
		cluster.ConnectTimeout = conn.Timeout
		cluster.Timeout = conn.Timeout
	}
	if conn.Port != 0 {
		cluster.Port = conn.Port
	}
	if conn.NumConns != 0 {
		cluster.NumConns = conn.NumConns
	}

	logger.BkLog.Infof("[cassandra] Connecting with config: %v", *conn)

	session, err := cluster.CreateSession()

	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cassandra - Could not connect database, details: %v", err))
		return nil, err
	}
	session.SetConsistency(gocql.One)
	return &BkCassandra{session}, nil
}

// Close -- close connection
func (c *BkCassandra) Close() {
	if c == nil {
		return
	}
	c.Session.Close()
}

// GetSesstion -- get session
func (c *BkCassandra) GetSesstion() *gocql.Session {
	return c.Session
}
