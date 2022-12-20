package bkmigration

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/punky97/go-codebase/core/logger"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

var mrx = regexp.MustCompile(`^(\d+)_([^.]+)(\.[a-z0-9]+)?\.(up|down)\.(ezm)$`)

func NewMigrator(c *Connection) Migrator {
	return Migrator{
		Connection: c,
		Migrations: map[string]Migrations{
			"up":   {},
			"down": {},
		},
	}
}

type Migrator struct {
	Connection    *Connection
	IncludeTables []string
	Migrations    map[string]Migrations
}

func (m *Migrator) SetIncludeTables(includeTables []string) {
	m.IncludeTables = includeTables
}

func (m *Migrator) Up() error {
	c := m.Connection
	return m.exec(func() error {
		mfs := m.Migrations["up"]
		sort.Sort(mfs)
		applied := 0
		for _, mi := range mfs {
			var err error
			if mi.DBType != "all" && mi.DBType != c.Dialect.Name() {
				// Skip migration for non-matching dialect
				continue
			}
			countRes := &rowCount{}
			query := fmt.Sprintf("SELECT COUNT(*) AS row_count FROM %s WHERE version = %s", m.Connection.MigrationTableName(), mi.Version)
			err = m.Connection.Store.Get(countRes, query)
			if countRes.Count > 0 {
				continue
			}

			err = c.Transaction(func(tx *Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return errors.New(fmt.Sprintf("Migration %s with version %s has run failed with error: %s", mi.Name, mi.Version, err))
				}
				query := fmt.Sprintf("INSERT INTO %s (table_name, migration_name, version) VALUES ('%s','%s','%s')", m.Connection.MigrationTableName(), mi.Table, mi.Name, mi.Version)
				fmt.Println(query)
				_, err = m.Connection.Store.Exec(query)
				return errors.Wrapf(err, "problem insert migration version %s for table %s with name %s", mi.Version, mi.Table, mi.Name)
			})
			if err != nil {
				return errors.WithStack(err)
			}

			logger.BkLog.Infof("> %s", mi.Name)
			applied++
		}
		if applied == 0 {
			logger.BkLog.Infof("Migrations already up to date, nothing to apply")
		}
		return nil
	})
}

func (m *Migrator) Down(step int) error {
	c := m.Connection
	return m.exec(func() error {
		var err error
		count, err := m.CountMigration()
		if err != nil {
			return errors.Wrap(err, "migration down: unable count existing migration")
		}

		mfs := m.Migrations["down"]
		sort.Sort(sort.Reverse(mfs))

		if len(mfs) > count {
			mfs = mfs[len(mfs)-count:]
		}

		if step > 0 && len(mfs) >= step {
			mfs = mfs[:step]
		}
		for _, mi := range mfs {
			countRes := &rowCount{}
			query := fmt.Sprintf("SELECT COUNT(*) AS row_count FROM %s WHERE version = %s", m.Connection.MigrationTableName(), mi.Version)
			err = m.Connection.Store.Get(countRes, query)
			if err != nil || countRes.Count == 0 {
				return errors.Wrapf(err, "problem checking for migration version %s", mi.Version)
			}
			err = c.Transaction(func(tx *Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return errors.New(fmt.Sprintf("Migration %s with version %s has run failed with error: %s", mi.Name, mi.Version, err))
				}
				query = fmt.Sprintf("DELETE FROM %s WHERE version = %s", m.Connection.MigrationTableName(), mi.Version)
				_, err = m.Connection.Store.Exec(query)
				return errors.Wrapf(err, "problem deleting migration version %s", mi.Version)
			})
			if err != nil {
				return err
			}
			logger.BkLog.Infof("< %s", mi.Name)
		}
		return nil
	})
}

func (m *Migrator) Reset() error {
	err := m.Down(-1)
	if err != nil {
		return errors.WithStack(err)
	}
	return m.Up()
}

func (m *Migrator) Status() error {
	err := m.CreateSchemaMigrations()
	if err != nil {
		return errors.WithStack(err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	_, _ = fmt.Fprintln(w, "Table\tVersion\tName\tStatus\t")
	for _, mf := range m.Migrations["up"] {
		exists := false
		err := m.Connection.Open()
		if err != nil {
			return errors.Wrapf(err, "error when open connection.")
		}
		res := &rowCount{}
		err = m.Connection.Store.Get(res, fmt.Sprintf("SELECT COUNT(*) AS row_count FROM %s WHERE version = %s",
			m.Connection.MigrationTableName(), mf.Version))
		if err != nil {
			return errors.Wrapf(err, "error when get migration information")
		}
		if res.Count > 0 {
			exists = true
		}
		state := "Pending"
		if exists {
			state = "Applied"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", mf.Table, mf.Version, mf.Name, state)
	}
	return w.Flush()
}

type rowCount struct {
	Count int `db:"row_count"`
}

func (m Migrator) CountMigration() (int, error) {
	res := &rowCount{}
	c := m.Connection
	err := c.Open()
	if err != nil {
		return 0, errors.Wrapf(err, "error when open connection.")
	}
	query := fmt.Sprintf("SELECT COUNT(*) AS row_count FROM %s", m.Connection.MigrationTableName())
	err = m.Connection.Store.Get(res, query)
	if err != nil {
		return 0, errors.Wrapf(err, "error when count migrations.")
	}
	return res.Count, nil
}

func (m Migrator) exec(fn func() error) error {
	now := time.Now()
	defer perfLog(now)
	err := m.Connection.Open()
	if err != nil {
		return errors.WithStack(err)
	}
	err = m.CreateSchemaMigrations()
	if err != nil {
		return errors.Wrap(err, "Migrator: problem creating schema migrations")
	}
	return fn()
}

func (m Migrator) CreateSchemaMigrations() error {
	c := m.Connection
	mtn := c.MigrationTableName()
	if c.Store == nil {
		err := c.Open()
		if err != nil {
			return errors.Wrap(err, "could not open connection")
		}
	}
	_, err := c.Store.Exec(fmt.Sprintf("select * from %s", mtn))
	if err == nil {
		return nil
	}

	return c.Transaction(func(tx *Connection) error {
		schemaMigrations := newSchemaMigrations(mtn)
		smSQL, err := c.Dialect.EzmTranslator().CreateTable(schemaMigrations)
		if err != nil {
			return errors.Wrap(err, "could not build SQL for schema migration table")
		}
		scripts := strings.Split(smSQL, ";")
		for _, s := range scripts {
			sql := strings.TrimSpace(s)
			if sql != "" {
				_, err = tx.Store.Exec(sql)
				if err != nil {
					return errors.Wrap(err, s)
				}
				fmt.Println(sql)
			}
		}
		return nil
	})
}

func perfLog(start time.Time) {
	delta := time.Since(start).Seconds()
	logger.BkLog.Infof("The process take `%v` seconds.", delta)
}
