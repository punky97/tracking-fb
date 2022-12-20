package mysql

import (
	"github.com/punky97/go-codebase/core/drivers/sqlhooks"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type SingleDBConnection struct {
	BaseConnection
	Host string
	Port int
}

func NewSingleConn(base BaseConnection, groupConfigArgs ...string) *SingleDBConnection {
	groupConfig := "mysql"
	if len(groupConfigArgs) > 0 {
		groupConfig = groupConfigArgs[0]
	}

	return &SingleDBConnection{
		BaseConnection: base,
		Host:           viper.GetString(fmt.Sprintf("%s.host", groupConfig)),
		Port:           viper.GetInt(fmt.Sprintf("%s.port", groupConfig)),
	}
}

func (conn *SingleDBConnection) BuildSetting() ([]string, []string) {
	settings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
		conn.User, conn.Password,
		conn.Protocol,
		conn.Host, conn.Port,
		conn.DatabaseName,
		conn.Charset, conn.ParseTime,
	)
	if utils.IsStringNotEmpty(conn.Others) {
		settings = fmt.Sprintf("%s&%s", settings, conn.Others)
	}

	logSettings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
		conn.User, utils.CensorString(conn.Password),
		conn.Protocol,
		conn.Host, conn.Port,
		conn.DatabaseName,
		conn.Charset, conn.ParseTime,
	)

	logger.BkLog.Info("[mysql] Connecting with this configuration: ", logSettings)

	return []string{settings}, nil
}

func (c *SingleDBConnection) GetHook() sqlhooks.Hooks {
	return c.Hooks
}

func (c *SingleDBConnection) GetMaxIdleConn() int {
	return c.MaxIdleConn
}

func (c *SingleDBConnection) GetMaxOpenConn() int {
	return c.MaxOpenConn
}
func (c *SingleDBConnection) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnectionLifeTime) * time.Second
}
