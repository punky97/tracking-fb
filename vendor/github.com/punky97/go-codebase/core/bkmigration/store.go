package bkmigration

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type store interface {
	Select(interface{}, string, ...interface{}) error
	Get(interface{}, string, ...interface{}) error
	NamedExec(string, interface{}) (sql.Result, error)
	Exec(string, ...interface{}) (sql.Result, error)
	PrepareNamed(string) (*sqlx.NamedStmt, error)
	Transaction() (*Tx, error)
	Rollback() error
	Commit() error
	Close() error
}