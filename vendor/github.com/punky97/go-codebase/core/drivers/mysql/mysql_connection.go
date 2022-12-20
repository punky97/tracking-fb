package mysql

import (
	"github.com/punky97/go-codebase/core/drivers/sqlhooks"
	"github.com/punky97/go-codebase/core/logger"
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"time"
)

// BkMysql --
type BkMysql struct {
	*DB
}

// NewMysqlConnection -- open connection to db
func NewConnection(conn Connection) (*BkMysql, error) {
	driverName := "mysql"
	if conn.GetHook() != nil {
		driverName := fmt.Sprintf("mysql-hook-%s", time.Now().String())
		sql.Register(driverName, sqlhooks.Wrap(&mysql.MySQLDriver{}, conn.GetHook()))
	}

	masterSetting, slaveSetting := conn.BuildSetting()
	db, err := Open(driverName, masterSetting, slaveSetting)

	if err != nil {
		logger.BkLog.Error("[mysql] Could not connect database, details: ", err)
		return nil, err
	}
	if conn.GetMaxOpenConn() > 0 {
		db.SetMaxOpenConns(conn.GetMaxOpenConn())
	}
	if conn.GetMaxIdleConn() > 0 && conn.GetMaxIdleConn() < conn.GetMaxOpenConn() {
		db.SetMaxIdleConns(conn.GetMaxIdleConn())
	}
	db.SetConnMaxLifetime(conn.GetConnMaxLifetime())

	err = db.Ping()
	if err != nil {
		logger.BkLog.Error("[mysql] Could not ping to database, details: ", err)
		return nil, err
	}

	return NewBkMysql(db)
}

func NewBkMysql(db *DB) (*BkMysql, error) {
	if db == nil {
		return nil, errors.New("nil db is provided")
	}

	return &BkMysql{
		DB: db,
	}, nil
}

func NewBkMysqlFromRawDB(db *sql.DB) (*BkMysql, error) {
	if db == nil {
		return nil, errors.New("nil db is provided")
	}

	return &BkMysql{
		DB: NewDB([]*SingleDB{NewSingleDB(db, 0)}, nil),
	}, nil
}

// Close -- close connection
func (c *BkMysql) Close() {
	if c == nil {
		return
	}

	if err := c.DB.Close(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when close db: %v", err))
	}
}

// GetDB -- get db
func (c *BkMysql) GetDB() *sql.DB {
	return c.DB.GetMaster()
}
