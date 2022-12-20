package mongo

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Connection -- Mongodb connection
type Connection struct {
	addresses    []string
	user         string
	password     string
	databaseName string
	direct       bool
	timeout      time.Duration
	failFast     bool
	poolLimit    int
	service      string
	serviceHost  string
	mechanism    string
}

// DefaultMongoConnectionFromConfig -- Load connection settings
func DefaultMongoConnectionFromConfig() *Connection {
	var host = viper.GetString("mongodb.host")
	var port = viper.GetInt("mongodb.port")

	var addr = fmt.Sprintf("%v:%v", host, port)

	var multiAddr = viper.GetStringSlice("mongodb.addresses")
	var address []string
	if len(multiAddr) == 0 {
		address = []string{addr}
	} else {
		address = multiAddr
	}

	return &Connection{
		addresses:    address,
		user:         viper.GetString("mongodb.user"),
		password:     viper.GetString("mongodb.password"),
		databaseName: viper.GetString("mongodb.database_name"),
		direct:       viper.GetBool("mongodb.direct"),
		timeout:      viper.GetDuration("mongodb.timeout"),
		failFast:     viper.GetBool("mongodb.fail_fast"),
		poolLimit:    viper.GetInt("mongodb.pool_limit"),
		service:      viper.GetString("mongodb.service"),
		serviceHost:  viper.GetString("mongodb.service_host"),
		mechanism:    viper.GetString("mongodb.mechanism"),
	}
}
