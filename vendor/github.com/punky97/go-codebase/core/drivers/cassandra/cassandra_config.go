package cassandra

import (
	"github.com/spf13/viper"
	"time"
)

// Connection -- mysql connection
type Connection struct {
	Cluster  []string
	Keyspace string
	Port     int
	NumConns int
	Timeout  time.Duration
}

// DefaultMysqlConnectionFromConfig -- load connection settings in config with default key
func DefaultCasssandraConnectionFromConfig() *Connection {
	return &Connection{
		Cluster:  viper.GetStringSlice("cassandra.clusters"),
		Port:     viper.GetInt("cassandra.port"),
		Keyspace: viper.GetString("cassandra.keyspace"),
		NumConns: viper.GetInt("cassandra.num_conns"),
		Timeout:  time.Duration(viper.GetInt("cassandra.timeout")) * time.Millisecond,
	}
}
