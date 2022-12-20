package sync

import (
	"github.com/cornelk/hashmap"
	"sync/atomic"
)

type IntIntMap struct {
	dataMap *hashmap.HashMap
}

func NewIntIntMap() *IntIntMap {
	return &IntIntMap{
		dataMap: &hashmap.HashMap{},
	}
}

func (m *IntIntMap) Increase(k int64, inc int64) {
	var i int64
	var count *int64
	var kptr = uintptr(k)
	if actual, ok := m.dataMap.GetUintKey(kptr); !ok {
		actual, _ := m.dataMap.GetOrInsert(kptr, &i)
		count = actual.(*int64)
	} else {
		count = actual.(*int64)
	}
	atomic.AddInt64(count, inc)
}

func (m *IntIntMap) GetOrInsert(k int64, val *int64) (*int64) {
	if actual, ok := m.dataMap.GetUintKey(uintptr(k)); ok {
		return actual.(*int64)
	} else {
		actual, _ := m.dataMap.GetOrInsert(uintptr(k), val)
		return actual.(*int64)
	}
}

func (m *IntIntMap) Get(k int64) (*int64, bool) {
	v, ok := m.dataMap.GetUintKey(uintptr(k))
	if v != nil {
		return v.(*int64), ok
	} else {
		return nil, false
	}
}

func (m *IntIntMap) Exist(k int64) bool {
	_, ok := m.dataMap.GetUintKey(uintptr(k))
	return ok
}

func (m *IntIntMap) Set(k int64, val *int64) {
	m.dataMap.Set(uintptr(k), val)
}

func (m *IntIntMap) Del(k int64) {
	m.dataMap.Del(k)
}

func (m *IntIntMap) Len() int {
	return m.dataMap.Len()
}

func (m *IntIntMap) Iter() <- chan hashmap.KeyValue {
	return m.dataMap.Iter()
}

func (m *IntIntMap) ToMap() map[int64]int64 {
	tm := make(map[int64]int64)
	for kv := range m.Iter() {
		tm[int64(kv.Key.(uintptr))] = atomic.LoadInt64(kv.Value.(*int64))
	}

	return tm
}