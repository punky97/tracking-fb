package sync

import (
	"github.com/cornelk/hashmap"
)

type HashMap struct {
	dataMap *hashmap.HashMap
}

func NewHashMap() *HashMap {
	return &HashMap{
		dataMap: &hashmap.HashMap{},
	}
}

func (m *HashMap) GetOrInsert(k string, val interface{}) interface{} {
	if actual, ok := m.dataMap.GetStringKey(k); ok {
		return actual
	} else {
		actual, _ = m.dataMap.GetOrInsert(k, val)
		return actual
	}
}

func (m *HashMap) Get(k string) (interface{}, bool) {
	return m.dataMap.GetStringKey(k)
}

func (m *HashMap) Exist(k string) bool {
	_, ok := m.dataMap.GetStringKey(k)
	return ok
}

func (m *HashMap) Set(k string, val interface{}) {
	m.dataMap.Set(k, val)
}

func (m *HashMap) Del(k string) {
	m.dataMap.Del(k)
}

func (m *HashMap) Len() int {
	return m.dataMap.Len()
}

func (m *HashMap) Iter() <- chan hashmap.KeyValue {
	return m.dataMap.Iter()
}