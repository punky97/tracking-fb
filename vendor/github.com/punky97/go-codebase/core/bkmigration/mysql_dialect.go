package bkmigration

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/punky97/go-codebase/core/bkmigration/translators"
	"strings"
)

const nameMySQL = "mysql"

var _ dialect = &mysql{}

func init() {
	AvailableDialects = append(AvailableDialects, nameMySQL)
	newConnection[nameMySQL] = newMySQL
}

type mysql struct {
	commonDialect
}

func newMySQL(conn *ConnectionDetails) (dialect, error) {
	cd := &mysql{
		commonDialect: commonDialect{ConnectionDetails: conn},
	}
	return cd, nil

}

func (m *mysql) Name() string {
	return nameMySQL
}

func (m *mysql) Info() *ConnectionDetails {
	return m.ConnectionDetails
}

func (m *mysql) EzmTranslator() translators.Translator {
	t := translators.NewMySQL(m.URL(), m.Info().Database)
	return t
}

func (m *mysql) URL() string {
	cd := m.ConnectionDetails
	if cd.URL != "" {
		return strings.TrimPrefix(cd.URL, "mysql://")
	}

	user := fmt.Sprintf("%s:%s@", cd.User, cd.Password)
	user = strings.Replace(user, ":@", "@", 1)
	if user == "@" || strings.HasPrefix(user, ":") {
		user = ""
	}

	addr := fmt.Sprintf("(%s:%s)", cd.Host, cd.Port)
	// in case of unix domain socket, tricky.
	// it is better to check Host is not valid inet address or has '/'.
	if cd.Port == "socket" {
		addr = fmt.Sprintf("unix(%s)", cd.Host)
	}
	s := "%s%s/%s?%s"
	return fmt.Sprintf(s, user, addr, cd.Database, cd.OptionsString(""))
}

func (m *mysql) MigrationURL() string {
	return m.URL()
}

func (m *mysql) Lock(fn func() error) error {
	return fn()
}

func (m *mysql) Details() *ConnectionDetails {
	return m.ConnectionDetails
}
