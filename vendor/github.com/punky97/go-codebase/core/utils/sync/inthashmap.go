package sync

import (
	"github.com/cornelk/hashmap"
	"sync/atomic"
)

type IntHashMap struct {
	dataMap *hashmap.HashMap
}

func NewIntHashMap() *IntHashMap {
	return &IntHashMap{&hashmap.HashMap{}}
}

func (m *IntHashMap) GetOrInsert(k string, val *int64) *int64 {
	if actual, ok := m.dataMap.GetStringKey(k); ok {
		return actual.(*int64)
	} else {
		actual, _ = m.dataMap.GetOrInsert(k, val)
		return actual.(*int64)
	}
}

func (m *IntHashMap) Get(k string) (*int64, bool) {
	v, ok := m.dataMap.GetStringKey(k)
	if v != nil {
		return v.(*int64), ok
	} else {
		return nil, false
	}
}

func (m *IntHashMap) Exist(k string) bool {
	_, ok := m.dataMap.GetStringKey(k)
	return ok
}

func (m *IntHashMap) Set(k string, val *int64) {
	m.dataMap.Set(k, val)
}

func (m *IntHashMap) Increase(k string, inc int64) {
	var i int64
	var count *int64
	if actual, ok := m.dataMap.GetStringKey(k); !ok {
		actual, _ := m.dataMap.GetOrInsert(k, &i)
		count = actual.(*int64)
	} else {
		count = actual.(*int64)
	}

	atomic.AddInt64(count, inc)
}

func (m *IntHashMap) Del(k string) {
	m.dataMap.Del(k)
}

func (m *IntHashMap) Len() int {
	return m.dataMap.Len()
}

func (m *IntHashMap) Iter() <-chan hashmap.KeyValue {
	return m.dataMap.Iter()
}

func (m *IntHashMap) ToMap() map[string]int64 {
	var mm = make(map[string]int64)
	for kv := range m.Iter() {
		mm[kv.Key.(string)] = atomic.LoadInt64(kv.Value.(*int64))
	}

	return mm
}