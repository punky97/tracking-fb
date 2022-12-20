package utils

import (
	"errors"
	"github.com/punky97/go-codebase/core/logger"
	"os"
	"path"
	"path/filepath"
)

// get closest directory contains Gopkg.toml
func GetGoRepo() string {
	if d, err := getClosestRepoDir(); d == "" {
		if err != nil {
			logger.BkLog.Infof("Err when get GOBKREPO: %s", err)
		}
		return os.Getenv("GOBKREPO")
	} else {
		return path.Base(d)
	}
}

func getClosestRepoDir() (string, error) {
	if wd, err := os.Getwd(); err != nil {
		return "", err
	} else {
		for {
			if _, err := os.Stat(path.Join(wd, "Gopkg.toml")); err == nil {
				return wd, nil
			}

			wd = filepath.Dir(wd)
			if wd == "/" {
				return "", errors.New("Cannot file closest bkrepo with Gopkg.toml")
			}
		}
	}
}
