package mysql

import (
	"github.com/punky97/go-codebase/core/drivers/sqlhooks"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type SinglePostgreDBConnection struct {
	BaseConnection
	Host string
	Port int
}

func (conn *SinglePostgreDBConnection) BuildSetting() ([]string, []string) {
	settings := fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable",
		conn.Host, conn.Port,
		conn.User, conn.Password,
		conn.DatabaseName,
	)
	if utils.IsStringNotEmpty(conn.Others) {
		settings = fmt.Sprintf("%s&%s", settings, conn.Others)
	}

	logSettings := fmt.Sprintf("postgres://%s:%s@%s:%v/%s?sslmode=disable",
		conn.User, utils.CensorString(conn.Password),
		conn.Host, conn.Port,
		conn.DatabaseName,
	)

	logger.BkLog.Info("[postgres] Connecting with this configuration: ", logSettings)

	return []string{settings}, nil
}

func (c *SinglePostgreDBConnection) GetHook() sqlhooks.Hooks {
	return c.Hooks
}

func (c *SinglePostgreDBConnection) GetMaxIdleConn() int {
	return c.MaxIdleConn
}

func (c *SinglePostgreDBConnection) GetMaxOpenConn() int {
	return c.MaxOpenConn
}
func (c *SinglePostgreDBConnection) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnectionLifeTime) * time.Second
}

// DefaultPostGreConnectionFromConfig -- load connection settings in config with default key
func DefaultPostGreConnectionFromConfig() Connection {
	maxOpenConn := viper.GetInt("postgre.max_open_conn")
	if maxOpenConn <= 0 {
		maxOpenConn = MaxOpenConn
	}

	maxIdleConn := viper.GetInt("postgre.max_idle_conn")
	if maxIdleConn <= 0 {
		maxIdleConn = MaxIdleConn
	}

	connLifeTime := viper.GetInt("postgre.conn_life_time")
	if connLifeTime <= 0 {
		connLifeTime = DefaultConnLifeTime
	}

	baseConn := BaseConnection{
		Protocol:           viper.GetString("postgre.protocol"),
		User:               viper.GetString("postgre.user"),
		Password:           viper.GetString("postgre.password"),
		DatabaseName:       viper.GetString("postgre.database_name"),
		MaxOpenConn:        maxOpenConn,
		MaxIdleConn:        maxIdleConn,
		ConnectionLifeTime: connLifeTime,
	}

	return &SinglePostgreDBConnection{
		BaseConnection: baseConn,
		Host:           viper.GetString("postgre.host"),
		Port:           viper.GetInt("postgre.port"),
	}
}
