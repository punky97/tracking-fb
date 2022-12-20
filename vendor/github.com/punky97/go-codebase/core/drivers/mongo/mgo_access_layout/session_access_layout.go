package mgo_access_layout

type SesstionAL interface {
	Close()
	Refresh()
	SetBatch(n int)
}