package mgo_mock

import "github.com/globalsign/mgo"

type Collection struct {
	Session *Session
	name    string
}

func (c *Collection) Update(selector interface{}, update interface{}) error {
	_, err := c.Session.matchUpdate(selector, update)
	return err
}
func (c *Collection) UpdateAll(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error) {
	_, updateInfo, err := c.Session.matchUpdateAll(selector, update)
	return updateInfo, err
}
func (c *Collection) Remove(selector interface{}) error {
	_, _, err := c.Session.matchRemove(selector, nil)
	return err
}
func (c *Collection) RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error) {
	_, changeInfo, err := c.Session.matchRemove(selector, nil)
	return changeInfo, err
}
func (c *Collection) Insert(docs ...interface{}) error {
	return c.Session.matchInsert(docs...)
}
func (c *Collection) Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error) {
	_, changeInfo, err := c.Session.matchUpserts(selector, update)
	return changeInfo, err
}
func (c *Collection) Find(query interface{}) *Query {
	return &Query{
		Session: c.Session,
		query: &QueryReal{
			collectionName: c.name,
			query:          query,
		},
	}
}
func (c *Collection) Pipe(pipeline interface{}) *Pipe {
	return nil
}
