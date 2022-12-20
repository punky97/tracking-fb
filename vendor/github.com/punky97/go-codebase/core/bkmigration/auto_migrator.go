package bkmigration

import (
	"encoding/json"
	"fmt"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type autoMigrate struct {
	connectionDetail ConnectionDetails
	includeTables    []string
	migrator         Migrator
	autoMigration    bool
	workingDirectory string
}

func RunAutoMigration() error {
	autoMigration := viper.GetBool("migration.auto_migration")
	if !autoMigration {
		logger.BkLog.Info("[Auto Migrate] Won't run because it wasn't enabled in configuration")
		return nil
	}

	cd := DefaultMySqlConnectionFromConfig()

	mig, err := newAutoMigrate(cd, autoMigration)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Initialize auto migrate failed with error: %s", err))
		return err
	}

	err = mig.autoMigrate()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Failed migrate with error: %s", err))
		return err
	}

	return nil
}

func newAutoMigrate(cd *ConnectionDetails, autoMigration bool) (*autoMigrate, error) {
	if cd == nil {
		logger.BkLog.Error("[Auto Migrate] Invalid mysql connection configuration")
		return nil, ErrInvalidDBConfiguration
	}
	conn, err := NewConnection(cd)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Failed initialize new connection with errors: %s", err))
		return nil, err
	}
	return &autoMigrate{
		connectionDetail: *cd,
		migrator:         NewMigrator(conn),
		autoMigration:    autoMigration,
	}, nil
}

func (m *autoMigrate) getMigrationPath(wd string) string {
	return fmt.Sprintf("%s/migrations/%s/%s", wd, m.connectionDetail.Dialect, m.connectionDetail.Database)
}

func (m *autoMigrate) autoMigrate() error {
	if !m.autoMigration {
		logger.BkLog.Info("[Auto Migrate] Won't run because it wasn't enabled in configuration")
		return nil
	}

	if err := m.setWorkingDirectory(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Cannot get working directory with err: %s", err))
		return err
	}

	wd, err := m.findRootPath()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] working directory is not root path: %v", err), "path", wd)
		return err
	}

	if err := m.findTablesMigrate(); err != nil {
		logger.BkLog.Infof("[Auto Migrate] cannot find tables need migrate with error: %s, auto-migration will be skipped", err)
		return nil
	}

	migrationPath := m.getMigrationPath(wd)
	paths := make([]string, 0)
	if len(m.includeTables) > 0 {
		for _, v := range m.includeTables {
			paths = append(paths, fmt.Sprintf("%s/%s", migrationPath, strings.TrimSpace(v)))
		}
	} else {
		paths = append(paths, migrationPath)
	}

	mig, err := NewFileMigrator(paths, m.migrator.Connection)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Failed new migrate file with errors: %s", err))
		return err
	}

	if err := mig.Up(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Failed applying new migration with details: %s", err))
		return err
	}

	if err := mig.Status(); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("[Auto Migrate] Cannot get status migrations with error: %s", err))
		return err
	}
	return nil
}

func (m *autoMigrate) setWorkingDirectory() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	m.workingDirectory = wd
	return nil
}

func (m *autoMigrate) findRootPath() (string, error) {
	paths := strings.Split(m.workingDirectory, "/")
	length := len(paths)
	currentPath := "/"
	for i := 0; i < length; i++ {
		currentPath += "/" + paths[i]
		if _, err := os.Stat(path.Join(currentPath, "migrations")); err == nil {
			return currentPath, nil
		} else {
			continue
		}
	}
	return m.workingDirectory, ErrNotFoundRootPath
}

type tablesMigrate struct {
	Table []string `json:"include_tables"`
}

func (m *autoMigrate) findTablesMigrate() error {
	fPath := path.Join(m.workingDirectory, "migrate.json")
	if fi, err := os.Stat(fPath); err != nil || fi.IsDir() {
		return err
	}
	f, err := os.Open(fPath)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	tm := &tablesMigrate{}
	if err := json.Unmarshal(data, tm); err != nil {
		return err
	}

	m.includeTables = tm.Table
	return nil
}
