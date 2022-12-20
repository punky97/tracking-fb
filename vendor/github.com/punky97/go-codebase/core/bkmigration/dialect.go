package bkmigration

import (
	"github.com/punky97/go-codebase/core/bkmigration/translators"
)

type dialect interface {
	Name() string
	URL() string
	Info() *ConnectionDetails
	MigrationURL() string
	Details() *ConnectionDetails
	Lock(func() error) error
	EzmTranslator() translators.Translator
}
