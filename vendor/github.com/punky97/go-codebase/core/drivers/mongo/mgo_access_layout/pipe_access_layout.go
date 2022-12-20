package mgo_access_layout

type PipeAL interface {
	All(result interface{}) error
	One(result interface{}) error
	Explain(result interface{}) error
}
