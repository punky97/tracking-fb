package mgo_access_layout

type QueryAL interface {
	One(result interface{}) (err error)
	All(result interface{}) error
}
