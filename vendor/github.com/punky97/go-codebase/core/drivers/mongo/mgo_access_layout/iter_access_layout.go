package mgo_access_layout

type IterAL interface {
	Next(result interface{}) bool
	Close() error
}