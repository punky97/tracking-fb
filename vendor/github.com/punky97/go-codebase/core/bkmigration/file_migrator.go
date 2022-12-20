package bkmigration

import (
	"bytes"
	"github.com/punky97/go-codebase/core/bkmigration/translators"
	"github.com/punky97/go-codebase/core/logger"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type FileMigrator struct {
	Migrator
	Path []string
}

func NewFileMigrator(paths []string, c *Connection) (FileMigrator, error) {
	fm := FileMigrator{
		Migrator: NewMigrator(c),
		Path:     paths,
	}
	err := fm.findMigrator()
	if err != nil {
		return fm, errors.WithStack(err)
	}
	return fm, nil
}

func (fm *FileMigrator) findMigrator() error {
	if !fm.isValidDirsPath() {
		return ErrInvalidDirsPath
	}
	for _, dir := range fm.Path {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			//check is dir
			if !info.IsDir() {
				matches := mrx.FindAllStringSubmatch(info.Name(), -1)
				if len(matches) == 0 {
					return nil
				}
				m := matches[0]
				var dbType string
				if m[3] == "" {
					dbType = "all"
				} else {
					dbType = m[3][1:]
					//check dialect supported
					if !DialectSupported(dbType) {
						return errors.Errorf("unsupported dialect: `%s`", dbType)
					}
				}
				paths := strings.Split(path, "/")
				table := paths[len(paths)-2]
				if table == "" {
					return errors.Errorf("Cannot apply migrate because not found table from dir path %s", dir)
				}
				mf := Migration{
					Path:      path,
					Table:     table,
					Version:   m[1],
					Name:      m[2],
					DBType:    dbType,
					Direction: m[4],
					Type:      m[5],
					Runner: func(mf Migration, conn *Connection) error {
						f, err := os.Open(path)
						if err != nil {
							return errors.WithStack(err)
						}

						query, err := migrationContent(mf, conn, f)
						if err != nil {
							return errors.WithStack(err)
						}

						if query == "" {
							return nil
						}

						isCreateTable := false
						if strings.Contains(strings.ToUpper(query), "CREATE TABLE") {
							isCreateTable = true
						}

						queries := strings.Split(query, ";")
						if len(queries) > 0 {
							if !isCreateTable {
								_, err = conn.Store.Exec("SET AUTOCOMMIT = 0")
								if err != nil {
									return err
								}
								_, err = conn.Store.Exec(fmt.Sprintf("LOCK TABLES %s WRITE, %s AS t READ", table, table))
								defer func() {
									_, err = conn.Store.Exec("COMMIT")
									if err != nil {
										logger.BkLog.Errorw(fmt.Sprintf("[Migrate] Cannot COMMIT with error: %s", err))
									}
									_, err = conn.Store.Exec("UNLOCK TABLES")
									if err != nil {
										logger.BkLog.Errorw(fmt.Sprintf("[Migrate] Cannot COMMIT with error: %s", err))
									}
								}()
								if err != nil {
									return err
								}
							}
							for _, v := range queries {
								fmt.Println(v)
								if v == "" {
									continue
								}
								_, err = conn.Store.Exec(v)
							}
							if err != nil {
								return err
							}
						}
						return nil
					},
				}
				fm.Migrations[mf.Direction] = append(fm.Migrations[mf.Direction], mf)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (fm *FileMigrator) isValidDirsPath() bool {
	for _, dir := range fm.Path {
		logger.BkLog.Infof("Dir path of migration folder: %s", dir)
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

// read migration file and translate to sql query
func migrationContent(mf Migration, conn *Connection, r io.Reader) (string, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return "", nil
	}
	t := template.Must(template.New("sql").Parse(string(b)))
	var bb bytes.Buffer
	err = t.Execute(&bb, conn.Dialect.Details())
	if err != nil {
		return "", errors.Wrapf(err, "could not execute migration template %s", mf.Path)
	}
	content := string(b)
	if mf.Type == "ezm" {
		content, err = translators.AString(content, conn.Dialect.EzmTranslator())
		if err != nil {
			return "", errors.Wrapf(err, "could not execute migration file `%s`", err)
		}
	}
	return content, nil
}
