package bkmigration

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Tx stores a transaction with an ID to keep track.
type Tx struct {
	ID int
	*sqlx.Tx
}

func newTX(db *dB) (*Tx, error) {
	t := &Tx{
		ID: rand.Int(),
	}
	tx, err := db.Beginx()
	t.Tx = tx
	return t, errors.Wrap(err, "could not create new transaction")
}

// Transaction simply returns the current transaction,
// this is defined so it implements the `store` interface.
func (tx *Tx) Transaction() (*Tx, error) {
	return tx, nil
}

// Close does nothing. This is defined so it implements the `store` interface.
func (tx *Tx) Close() error {
	return nil
}
