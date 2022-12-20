package mgo_access_layout

import "github.com/globalsign/mgo"

type CollectionAL interface {
	Update(selector interface{}, update interface{}) error
	UpdateAll(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	Remove(selector interface{}) error
	RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error)
	Insert(docs ...interface{}) error
	Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
}
