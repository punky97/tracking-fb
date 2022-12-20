package mysql

import (
	"fmt"
	"github.com/punky97/go-codebase/core/drivers/sqlhooks"
	"github.com/spf13/viper"
	"time"
)

type Connection interface {
	BuildSetting() ([]string, []string)
	GetHook() sqlhooks.Hooks
	GetMaxIdleConn() int
	GetMaxOpenConn() int
	GetConnMaxLifetime() time.Duration
}

// Connection -- mysql connection
type BaseConnection struct {
	Protocol           string
	User               string
	Password           string
	DatabaseName       string
	Charset            string
	ParseTime          bool
	Others             string
	MaxOpenConn        int
	MaxIdleConn        int
	ConnectionLifeTime int
	Hooks              sqlhooks.Hooks
}

const (
	// MaxOpenConn --
	MaxOpenConn = 100
	// MaxIdleConn --
	MaxIdleConn = 10
	// ConnectionLifeTime --
	DefaultConnLifeTime = 600

	// ConnType
	ConnSingle      = "single"
	ConnMasterSlave = "master_slave"
)

// DefaultMysqlConnectionFromConfig -- load connection settings in config with default key
func DefaultMysqlConnectionFromConfig() Connection {
	return DefaultMySqlConnectionFromConfigWithHooks(nil)
}

// DefaultMySqlConnectionFromConfigWithHooks - load connection settings in config with default key and customized hooks
func DefaultMySqlConnectionFromConfigWithHooks(hooks sqlhooks.Hooks) Connection {
	return CreateConnectionConfigFromConfigGroupWithHooks("mysql", hooks)
}

func CreateConnectionConfigFromConfigGroupWithHooks(configGroup string, hooks sqlhooks.Hooks) Connection {
	maxOpenConn := viper.GetInt(fmt.Sprintf("%s.max_open_conn", configGroup))
	if maxOpenConn <= 0 {
		maxOpenConn = MaxOpenConn
	}

	maxIdleConn := viper.GetInt(fmt.Sprintf("%s.max_idle_conn", configGroup))
	if maxIdleConn <= 0 {
		maxIdleConn = MaxIdleConn
	}

	connLifeTime := viper.GetInt(fmt.Sprintf("%s.conn_life_time", configGroup))
	if connLifeTime <= 0 {
		connLifeTime = DefaultConnLifeTime
	}

	baseConfig := BaseConnection{
		Protocol:           viper.GetString(fmt.Sprintf("%s.protocol", configGroup)),
		User:               viper.GetString(fmt.Sprintf("%s.user", configGroup)),
		Password:           viper.GetString(fmt.Sprintf("%s.password", configGroup)),
		DatabaseName:       viper.GetString(fmt.Sprintf("%s.database_name", configGroup)),
		Charset:            viper.GetString(fmt.Sprintf("%s.charset", configGroup)),
		ParseTime:          viper.GetBool(fmt.Sprintf("%s.parse_time", configGroup)),
		Others:             viper.GetString(fmt.Sprintf("%s.others", configGroup)),
		MaxOpenConn:        maxOpenConn,
		MaxIdleConn:        maxIdleConn,
		ConnectionLifeTime: connLifeTime,
		Hooks:              hooks,
	}

	connType := viper.GetString(fmt.Sprintf("%s.conn_type", configGroup))

	switch connType {
	case ConnMasterSlave:
		return NewMasterSlaveConn(baseConfig)
	default:
		return NewSingleConn(baseConfig, configGroup)
	}
}
