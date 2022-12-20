package local

import (
	"github.com/punky97/go-codebase/core/cache/codec"
	"github.com/punky97/go-codebase/core/cache/lru"
	"sync"
	"time"
)

const (
	CacheCodeHit  = 1
	CacheCodeMiss = 0

	CacheCodeExpired = -1
	CacheCodeError   = -2

	_maxLocalCacheTime = time.Hour * 24
)

type BkLocalCache struct {
	lock       sync.RWMutex
	localCache *lru.Cache
}

type LocalCacheItem struct {
	V   interface{} `msgpack:"v"`
	Ct  int64       `msgpack:"ct"`  // created time
	Ttl int64       `msgpack:"ttl"` // expired time
}

func newLocalCacheItem(v interface{}, expireTime time.Duration) *LocalCacheItem {
	return &LocalCacheItem{
		V:   v,
		Ct:  time.Now().UnixNano(),
		Ttl: expireTime.Nanoseconds(),
	}
}

func (i *LocalCacheItem) expired() bool {
	return time.Now().UnixNano()-i.Ct > i.Ttl
}

func NewLocalCache(cacheSize int) *BkLocalCache {
	return &BkLocalCache{
		localCache: lru.New(cacheSize),
	}
}

func (c *BkLocalCache) Get(key string) (interface{}, int, error) {
	c.lock.Lock()
	tmp, cacheExist := c.localCache.Get(key)
	c.lock.Unlock()

	if !cacheExist {
		return nil, CacheCodeMiss, nil
	}

	item := &LocalCacheItem{}
	err := codec.DecodeToStruct(tmp.([]byte), item)
	if err != nil {
		return nil, CacheCodeError, err
	}

	if item.expired() {
		c.lock.Lock()
		c.localCache.Remove(key)
		c.lock.Unlock()

		return nil, CacheCodeExpired, nil
	} else {
		return item.V, CacheCodeHit, nil
	}
}

func (c *BkLocalCache) Set(key string, v interface{}, ttl time.Duration) error {
	if ttl > _maxLocalCacheTime {
		ttl = _maxLocalCacheTime
	}

	item := newLocalCacheItem(v, ttl)

	valueB, err := codec.Encode(item)
	if err != nil {
		return err
	}

	c.lock.Lock()
	c.localCache.Add(key, valueB)
	c.lock.Unlock()

	return nil
}

func (c *BkLocalCache) Del(key string) {
	c.lock.Lock()
	_, cacheExist := c.localCache.Get(key)
	c.lock.Unlock()

	if cacheExist {
		c.lock.Lock()
		c.localCache.Remove(key)
		c.lock.Unlock()
	}
}
