package bkmigration

type commonDialect struct {
	ConnectionDetails *ConnectionDetails
}

func (commonDialect) Lock(fn func() error) error {
	return fn()
}

