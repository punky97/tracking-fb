package mgo_mock

type Database struct {
	Session *Session
}

func (db *Database) Run(cmd interface{}, result interface{}) error {
	return nil
}

func (db *Database) C(name string) *Collection {
	return &Collection{name: name, Session: db.Session}
}
