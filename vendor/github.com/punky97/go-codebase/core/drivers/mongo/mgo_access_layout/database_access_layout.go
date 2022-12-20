package mgo_access_layout

type DatabaseAL interface {
	Run(cmd interface{}, result interface{}) error
}
