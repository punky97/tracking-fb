package mysql

import (
	"github.com/punky97/go-codebase/core/drivers/sqlhooks"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type MasterSlaveDBConnection struct {
	BaseConnection
	MasterAddresses []string
	SlaveAddresses  []string
}

func NewMasterSlaveConn(base BaseConnection) *MasterSlaveDBConnection {
	return &MasterSlaveDBConnection{
		BaseConnection:  base,
		MasterAddresses: viper.GetStringSlice("mysql.master_addresses"),
		SlaveAddresses:  viper.GetStringSlice("mysql.slave_addresses"),
	}
}

func (c *MasterSlaveDBConnection) BuildSetting() ([]string, []string) {
	listMasterSetting := make([]string, 0)
	for _, addr := range c.MasterAddresses {
		host, port := utils.ParseAddr(addr)
		settings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
			c.User, c.Password,
			c.Protocol,
			host, port,
			c.DatabaseName,
			c.Charset, c.ParseTime,
		)
		if utils.IsStringNotEmpty(c.Others) {
			settings = fmt.Sprintf("%s&%s", settings, c.Others)
		}
		listMasterSetting = append(listMasterSetting, settings)

		logSettings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
			c.User, utils.CensorString(c.Password),
			c.Protocol,
			host, port,
			c.DatabaseName,
			c.Charset, c.ParseTime,
		)

		logger.BkLog.Info("[mysql] Connecting with this configuration: ", logSettings)
	}

	listSlaveSetting := make([]string, 0)
	for _, addr := range c.SlaveAddresses {
		host, port := utils.ParseAddr(addr)
		settings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
			c.User, c.Password,
			c.Protocol,
			host, port,
			c.DatabaseName,
			c.Charset, c.ParseTime,
		)
		if utils.IsStringNotEmpty(c.Others) {
			settings = fmt.Sprintf("%s&%s", settings, c.Others)
		}
		listSlaveSetting = append(listSlaveSetting, settings)

		logSettings := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?charset=%s&parseTime=%t",
			c.User, utils.CensorString(c.Password),
			c.Protocol,
			host, port,
			c.DatabaseName,
			c.Charset, c.ParseTime,
		)

		logger.BkLog.Info("[mysql] Connecting with this configuration: ", logSettings)
	}

	return listMasterSetting, listSlaveSetting
}

func (c *MasterSlaveDBConnection) GetHook() sqlhooks.Hooks {
	return c.Hooks
}
func (c *MasterSlaveDBConnection) GetMaxIdleConn() int {
	return c.MaxIdleConn
}

func (c *MasterSlaveDBConnection) GetMaxOpenConn() int {
	return c.MaxOpenConn
}
func (c *MasterSlaveDBConnection) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnectionLifeTime) * time.Second
}
