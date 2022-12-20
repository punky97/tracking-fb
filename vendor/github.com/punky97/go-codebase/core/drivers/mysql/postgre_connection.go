package mysql

import (
	"database/sql"
	"github.com/punky97/go-codebase/core/logger"

	// postgre driver
	_ "github.com/lib/pq"
)

// NewPostgreConnection -- open connection to db
func NewPostgreConnection(conn Connection) (*BkMysql, error) {
	var err error

	settings, _ := conn.BuildSetting()

	db, err := sql.Open("postgres", settings[0])
	if err != nil {
		logger.BkLog.Error("[postgres] Could not connect database, details: ", err)
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
		logger.BkLog.Error("[postgres] Could not ping to database, details: ", err)
		return nil, err
	}

	return NewBkMysqlFromRawDB(db)
}
