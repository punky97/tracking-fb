package bkmigration

import (
	"github.com/punky97/go-codebase/core/bkmigration/utils/x/randx"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	// MaxOpenConn --
	MaxOpenConn = 100
	// MaxIdleConn --
	MaxIdleConn = 10
)

var Connections = map[string]*Connection{}

type Connection struct {
	ID          string
	Database    string
	Store       store
	Dialect     dialect
	TX          *Tx
	eager       bool
	eagerFields []string
}

func (conn *Connection) String() string {
	return conn.URL()
}

func (conn *Connection) URL() string {
	return conn.Dialect.URL()
}

func (conn *Connection) MigrationURL() string {
	return conn.Dialect.MigrationURL()
}

func (conn *Connection) MigrationTableName() string {
	return conn.Dialect.Info().MigrationTableName()
}

func NewConnection(cd *ConnectionDetails) (*Connection, error) {
	err := cd.Finalize()
	if err != nil {
		return nil, err
	}
	c := &Connection{
		ID: randx.String(30),
	}

	if nc, ok := newConnection[cd.Dialect]; ok {
		c.Dialect, err = nc(cd)
		if err != nil {
			return c, errors.Wrap(err, "could not create new connection")
		}
		c.Database = cd.Database
		return c, nil
	}

	return nil, errors.Errorf("could not found connection creator for `%s`", c.Dialect)
}

func (conn *Connection) Open() error {
	var err error

	if conn.Store != nil {
		return nil
	}

	if conn.Dialect == nil {
		return errors.New("invalid connection instance.")
	}

	info := conn.Dialect.Info()
	db, err := sqlx.Open(info.Dialect, conn.Dialect.URL())
	if err != nil {
		return errors.Wrap(err, "could not open database connection")
	}

	if info.IdlePool <= 0 {
		info.IdlePool = MaxIdleConn
	}

	if info.Pool <= 0 {
		info.Pool = MaxOpenConn
	}

	db.SetMaxIdleConns(info.IdlePool)
	db.SetMaxOpenConns(info.Pool)
	conn.Store = &dB{db}

	return nil
}

func (conn *Connection) Close() error {
	return errors.Wrap(conn.Store.Close(), "couldn't close connection")
}

func (conn *Connection) Transaction(fn func(tx *Connection) error) error {
	return conn.Dialect.Lock(func() error {
		var dberr error
		cn, err := conn.NewTransaction()
		if err != nil {
			return err
		}
		defer cn.Close()

		err = fn(cn)
		if err != nil {
			dberr = cn.TX.Rollback()
		} else {
			dberr = cn.TX.Commit()
		}

		if err != nil {
			return errors.WithStack(err)
		}
		return errors.Wrap(dberr, "error committing or rolling back transaction")
	})
}

func (conn *Connection) Rollback(fn func(tx *Connection)) error {
	cn, err := conn.NewTransaction()
	if err != nil {
		return err
	}
	fn(cn)
	return cn.TX.Rollback()
}

func (conn *Connection) NewTransaction() (*Connection, error) {
	var cn *Connection
	if conn.TX == nil {
		tx, err := conn.Store.Transaction()
		if err != nil {
			return cn, errors.Wrap(err, "couldn't start a new transaction")
		}
		cn = &Connection{
			ID:      randx.String(30),
			Store:   tx,
			Dialect: conn.Dialect,
			TX:      tx,
		}
	} else {
		cn = conn
	}
	return cn, nil
}

func (conn *Connection) copy() *Connection {
	return &Connection{
		ID:      randx.String(30),
		Store:   conn.Store,
		Dialect: conn.Dialect,
		TX:      conn.TX,
	}
}
