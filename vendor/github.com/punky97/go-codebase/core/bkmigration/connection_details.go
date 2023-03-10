package bkmigration

import (
	"github.com/punky97/go-codebase/core/bkmigration/utils/x/defaults"
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/url"
	"regexp"
	"strings"
)

type ConnectionDetails struct {
	// Example: "postgres" or "mysql"
	Dialect string
	// The name of your database. Example: "foo_development"
	Database string
	// The host of your database. Example: "127.0.0.1"
	Host string
	// The port of your database. Example: 3306
	// Will default to the "default" port for each dialect.
	Port string
	// The username of the database user. Example: "root"
	User string
	// The password of the database user. Example: "password"
	Password string
	// The encoding to use to create the database and communicate with it.
	Encoding string
	// Instead of specifying each individual piece of the
	// connection you can instead just specify the URL of the
	URL string
	// Defaults to 0 "unlimited". See https://golang.org/pkg/database/sql/#DB.SetMaxOpenConns
	Pool int
	// Defaults to 0 "unlimited". See https://golang.org/pkg/database/sql/#DB.SetMaxIdleConns
	IdlePool   int
	Options    map[string]string
	RawOptions string
}

var dialectX = regexp.MustCompile(`\S+://`)

// withURL parses and overrides all connection details with values
// from standard URL except Dialect. It also calls dialect specific
// URL parser if exists.
func (cd *ConnectionDetails) withURL() error {
	ul := cd.URL
	if cd.Dialect == "" {
		if dialectX.MatchString(ul) {
			// Guess the dialect from the scheme
			dialect := ul[:strings.Index(ul, ":")]
			cd.Dialect = normalizeSynonyms(dialect)
		} else {
			return errors.New("no dialect provided, and could not guess it from URL")
		}
	} else if !dialectX.MatchString(ul) {
		ul = cd.Dialect + "://" + ul
	}

	if !DialectSupported(cd.Dialect) {
		return errors.Errorf("unsupported dialect '%s'", cd.Dialect)
	}

	// warning message is required to prevent confusion
	// even though this behavior was documented.
	if cd.Database+cd.Host+cd.Port+cd.User+cd.Password != "" {
		logger.BkLog.Info("One or more of connection details are specified in database.yml. Override them with values in URL.")
	}

	if up, ok := urlParser[cd.Dialect]; ok {
		return up(cd)
	}

	// Fallback on generic parsing if no URL parser was found for the dialect.
	u, err := url.Parse(ul)
	if err != nil {
		return errors.Wrapf(err, "couldn't parse %s", ul)
	}
	cd.Database = strings.TrimPrefix(u.Path, "/")

	hp := strings.Split(u.Host, ":")
	cd.Host = hp[0]
	if len(hp) > 1 {
		cd.Port = hp[1]
	}

	if u.User != nil {
		cd.User = u.User.Username()
		cd.Password, _ = u.User.Password()
	}
	cd.RawOptions = u.RawQuery

	return nil
}

// Finalize cleans up the connection details by normalizing names,
// filling in default values, etc...
func (cd *ConnectionDetails) Finalize() error {
	cd.Dialect = normalizeSynonyms(cd.Dialect)

	if cd.Options == nil { // for safety
		cd.Options = make(map[string]string)
	}

	// Process the database connection string, if provided.
	if cd.URL != "" {
		if err := cd.withURL(); err != nil {
			return err
		}
	}

	if fin, ok := finalizer[cd.Dialect]; ok {
		fin(cd)
	}

	if DialectSupported(cd.Dialect) {
		if cd.Database != "" || cd.URL != "" {
			return nil
		}
		return errors.New("no database or URL specified")
	}
	return errors.Errorf("unsupported dialect '%v'", cd.Dialect)
}

// MigrationTableName returns the name of the table to track migrations
func (cd *ConnectionDetails) MigrationTableName() string {
	return defaults.String(cd.Options["migration_table_name"], "db_migrations")
}

// OptionsString returns URL parameter encoded string from options.
func (cd *ConnectionDetails) OptionsString(s string) string {
	if cd.RawOptions != "" {
		return cd.RawOptions
	}
	if cd.Options != nil {
		for k, v := range cd.Options {
			s = fmt.Sprintf("%s&%s=%s", s, k, v)
		}
	}
	return strings.TrimLeft(s, "&")
}

func DefaultMySqlConnectionFromConfig() *ConnectionDetails {
	return &ConnectionDetails{
		Dialect:  "mysql",
		Host:     viper.GetString("mysql.host"),
		Port:     viper.GetString("mysql.port"),
		User:     viper.GetString("mysql.user"),
		Password: viper.GetString("mysql.password"),
		Database: viper.GetString("mysql.database_name"),
	}
}