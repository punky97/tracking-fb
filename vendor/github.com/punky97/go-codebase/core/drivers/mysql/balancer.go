package mysql

import (
	"database/sql"
	"sync/atomic"
)

type DbBalancer interface {
	Next() (*sql.DB, int)
	NextSlave() (*sql.DB, int)
	NextMaster() (*sql.DB, int)
}

type rrBalancer struct {
	dbs         []*SingleDB
	masters     []*SingleDB
	slaves      []*SingleDB
	countMaster uint32
	countSlave  uint32
}

func RoundRobin(masters []*SingleDB, slaves []*SingleDB) *rrBalancer {
	dbs := make([]*SingleDB, 0)
	dbs = append(dbs, masters...)
	dbs = append(dbs, slaves...)
	return &rrBalancer{
		dbs:         dbs,
		masters:     masters,
		slaves:      slaves,
		countSlave:  0,
		countMaster: 0,
	}
}

func (b *rrBalancer) Next() (*sql.DB, int) {
	v1 := atomic.LoadUint32(&b.countMaster)
	v2 := atomic.LoadUint32(&b.countSlave)
	idx := (v1 + v2) % (uint32(len(b.dbs)))
	atomic.AddUint32(&b.countSlave, 1)

	_db := b.dbs[idx]
	return _db.DB, int(_db.id)
}

func (b *rrBalancer) NextMaster() (*sql.DB, int) {
	v := atomic.LoadUint32(&b.countMaster)
	idx := v % (uint32(len(b.masters)))
	atomic.AddUint32(&b.countMaster, 1)
	_db := b.masters[idx]
	return _db.DB, int(_db.id)
}

func (b *rrBalancer) NextSlave() (*sql.DB, int) {
	if len(b.slaves) == 0 {
		return b.NextMaster()
	} else {
		v := atomic.LoadUint32(&b.countSlave)
		idx := v % (uint32(len(b.slaves)))
		atomic.AddUint32(&b.countSlave, 1)
		_db := b.slaves[idx]
		return _db.DB, int(_db.id)
	}
}
