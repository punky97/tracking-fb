package bkmigration

import "errors"

var (
	ErrConfigFileNotFound = errors.New("unable find config file.")
	ErrInvalidDBConfiguration = errors.New("database connection configuration is invalid")
	ErrInvalidDirsPath = errors.New("invalid dirs path")
	ErrNotFoundRootPath = errors.New("cannot find root of working directory to get migrations folder, the root must contains migrations folder")
)
