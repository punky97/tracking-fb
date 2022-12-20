package cache

import (
	"github.com/punky97/go-codebase/core/cache/codec"
	"github.com/punky97/go-codebase/core/cache/lru"
	"github.com/punky97/go-codebase/core/drivers/bkredis"
	"github.com/punky97/go-codebase/core/logger"
	"github.com/punky97/go-codebase/core/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"

	"github.com/golang/protobuf/proto"
)

const (
	// MaxLocalCacheSize max allowed size of the local cache
	MaxLocalCacheSize = 10000
	// DefaultRemoteCacheTimeout timeout for redis cache
	DefaultRemoteCacheTimeout = 6 * 60
	// DefaultRemoteCacheTimeoutPeriod --
	DefaultRemoteCacheTimeoutPeriod = int64(6 * time.Minute)
	// DefaultLocalCacheTimeoutPeriod -
	DefaultLocalCacheTimeoutPeriod = int64(20 * time.Second)
	// DefaultMaxDayPeriod > 10 days cache will be rejected
	DefaultMaxDayPeriod = 1000000
	// CacheHitLocal hit from local level
	CacheHitLocal = 1
	// CacheHitRemote hit from remote level
	CacheHitRemote = 2
	// CacheMiss cache miss
	CacheMiss = 0
	// CacheError cannot get data from cache
	CacheError = -1
	// CacheExpired --
	CacheExpired = -2
	// WarnThreshold -
	WarnThreshold = 100 * time.Millisecond
	// BufferChanSize Buffer channel size
	BufferChanSize = 10000
)

//
var (
	WarnLogMode  = true
	DebugLogMode = false
)

// BkCacheStruct struct
type BkCacheStruct struct {
	AppName             string
	LocalCache          *lru.Cache
	Redis               *bkredis.RedisClient
	RedisFeatured       bool
	LocalFeatured       bool // enable local cache
	SlowCacheReportMode bool
	lock                sync.RWMutex
	del                 chan string
	sets                chan *setData
	lastCtx             context.Context
	cancelFuncs         []func()
	cacheSyncEnabled    bool

	// status
	hitLocal          uint64
	hitRemote         uint64
	total             uint64
	totalSet          uint64
	setRemoteCount    uint64
	miss              uint64
	expired           uint64
	err               uint64
	delCount          uint64
	slowCacheGetCount int64
	slowCacheSetCount int64
	getTime           uint64
}

// CachedItem --
type CachedItem struct {
	V   interface{} `msgpack:"v"`
	Ct  int64       `msgpack:"ct"`  // created time
	LDt int64       `msgpack:"ldt"` // durable time
	RDt int64       `msgpack:"rdt"` // durable time
}

func (i *CachedItem) expired(hitMode int) bool {
	if hitMode <= 0 {
		return true
	}

	switch hitMode {
	case CacheHitLocal:
		return time.Now().UnixNano()-i.Ct > i.LDt
	case CacheHitRemote:
		return time.Now().UnixNano()-i.Ct > i.RDt
	}
	return true
}

// BkCache exports for other classes to use
var BkCache *BkCacheStruct
var lock sync.Mutex

func init() {
	lock.Lock()
	defer lock.Unlock()
	if BkCache == nil {
		BkCache = New()
	}
}

// New --
func New() *BkCacheStruct {
	bc := &BkCacheStruct{
		LocalCache:       lru.New(MaxLocalCacheSize),
		RedisFeatured:    false,
		LocalFeatured:    true,
		cacheSyncEnabled: false,
		sets:             make(chan *setData, 100),
		del:              make(chan string, 1),
	}
	ctx, cancelFunc := utils.GetContextWithCancelSafe2()
	bc.lastCtx = ctx
	bc.cancelFuncs = []func(){cancelFunc}

	// setup watcher
	newCtx := utils.GetContextWithRandomKV(bc.lastCtx)
	go func(ctx context.Context) {
		tick := time.NewTicker(time.Minute * 2)

		var lastTime = time.Now()
		var lastTotal, lastTotalSet, lastHitLocal, lastHitRemote, lastMiss, lastErr, lastExpired uint64
		var lastDelCount, lastSetCount, lastGetTime uint64

		for {
			select {
			case <-tick.C:
				var now = time.Now()
				var mapSize = bc.LocalCache.MapSize()
				var listSize = bc.LocalCache.ListSize()
				var total = bc.total
				var totalSet = bc.totalSet
				var hitLocal = bc.hitLocal
				var hitRemote = bc.hitRemote
				var miss = bc.miss
				var cerr = bc.err
				var expired = bc.expired
				var delCount = bc.delCount
				var setCount = bc.setRemoteCount
				var getTime = bc.getTime
				var meanTime, meanTimeTotal = 0.0, 0.0
				if total-lastTotal > 0 {
					meanTime = float64(getTime-lastGetTime) / float64(total-lastTotal)
				}
				if total > 0 {
					meanTimeTotal = float64(getTime) / float64(total)
				}

				if DebugLogMode {
					logger.BkLog.Infof(`Current cache status: map items(%v), list items(%v), 
total_cache_request(%v/%v, %v/%v), hit_local(%v/%v), hit_remote(%v/%v), miss(%v/%v), err(%v/%v), expired(%v/%v), 
del_count(%v/%v), set_remote_count(%v/%v), mean_time_ms(%.3f/%.3f) in %v`,
						mapSize, listSize,
						total-lastTotal, total, totalSet-lastTotalSet, totalSet, hitLocal-lastHitLocal, hitLocal, hitRemote-lastHitRemote, hitRemote,
						miss-lastMiss, miss, cerr-lastErr, cerr, expired-lastExpired, expired,
						delCount-lastDelCount, delCount, setCount-lastSetCount, setCount,
						meanTime, meanTimeTotal, now.Sub(lastTime))
				}
				lastTotal = total
				lastTotalSet = totalSet
				lastHitLocal = hitLocal
				lastHitRemote = hitRemote
				lastMiss = miss
				lastErr = cerr
				lastExpired = expired
				lastDelCount = delCount
				lastSetCount = setCount
				lastGetTime = getTime
				lastTime = now

			case <-newCtx.Done():
				return
			default:
				time.Sleep(60 * time.Millisecond)
			}
		}
	}(newCtx)

	return bc
}

type setData struct {
	key   string
	value interface{}
	ttl   int
}

func (bc *BkCacheStruct) SetCacheConfig() {
	bc.LocalFeatured = viper.GetBool("cache.enable_local_cache")
	bc.cacheSyncEnabled = viper.GetBool("cache.enable_cache_sync")
	bc.SlowCacheReportMode = viper.GetBool("cache.slow_cache_report")
}

// SetEnableLocal enable or disable local cache
func (bc *BkCacheStruct) SetEnableLocal(enable bool) {
	bc.LocalFeatured = enable
}

// SetCacheSyncEnabled -
func (bc *BkCacheStruct) SetCacheSyncEnabled(enable bool) {
	bc.cacheSyncEnabled = enable
}

func (bc *BkCacheStruct) setLoop(ctx context.Context) {
	logger.BkLog.Debugf("Start set loop")
loop_set:
	for {
		select {
		case <-ctx.Done():
			break loop_set
		case s := <-bc.sets:
			if s == nil {
				continue
			}
			res := bc.Redis.Set(ctx, s.key, s.value, time.Duration(s.ttl)*time.Second)
			if res.Err() != nil {
				logger.BkLog.Warn("Cannot set redis key. ", res.Err())
			} else {
				atomic.AddUint64(&bc.setRemoteCount, 1)
			}
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// Clear clear all local cache and prefix cache in redis
func (bc *BkCacheStruct) Clear(ctx context.Context, pattern string) {
	bc.lock.Lock()
	bc.LocalCache.Clear()
	bc.lock.Unlock()
	if bc.RedisFeatured {
		clearClientKey := func(redisClient redis.UniversalClient) {
			var cursor uint64
			var ks []string
			var err error

			cmd := redisClient.Scan(ctx, cursor, pattern, 1000)
			go func() {
				c := 0
			loop:
				for {
					c++
					if ks, cursor, err = cmd.Result(); len(ks) > 0 && err == nil {
						redisClient.Del(ctx, ks...)
						if bc.cacheSyncEnabled {
							for _, k := range ks {
								now := time.Now()
								// flush cache on all node
								data, err := json.Marshal(cacheMessage{
									Sec:  now.Unix(),
									Nano: now.Nanosecond(),
									Key:  k,
								})
								if err != nil {
									logger.BkLog.Errorw("Error when serialize cache message", "error", err)
								} else {
									err := bc.Redis.Publish(ctx, bc.getRedisChanName(), string(data)).Err()
									if err != nil {
										logger.BkLog.Warn("Error when push to redis chan ", bc.getRedisChanName())
									}
								}
							}
						}

						cmd = redisClient.Scan(ctx, cursor, pattern, 1000)
					} else if cursor != 0 {
						cmd = redisClient.Scan(ctx, cursor, pattern, 1000)
					} else {
						logger.BkLog.Info("Exit round: ", c, cursor, ks, err)
						break loop
					}
				}

				logger.BkLog.Debug("All cached data is cleared for node ", redisClient)
			}()
		}

		switch bc.Redis.UniversalClient.(type) {
		case *redis.ClusterClient:
			clusterClient, _ := bc.Redis.UniversalClient.(*redis.ClusterClient)
			err := clusterClient.ForEachShard(ctx, func(ctx context.Context, client *redis.Client) error {
				clearClientKey(client)
				return nil
			})

			if err != nil {
				logger.BkLog.Errorw("Foreach shard error", "error", err.Error())
			}
		default:
			clearClientKey(bc.Redis.UniversalClient)
		}
	}

	bc.err = 0
	bc.expired = 0
	bc.hitLocal = 0
	bc.hitRemote = 0
	bc.total = 0
	bc.totalSet = 0
	bc.miss = 0
	bc.setRemoteCount = 0
	bc.delCount = 0
	bc.getTime = 0
	bc.slowCacheGetCount = 0
	bc.slowCacheSetCount = 0
}

func (bc *BkCacheStruct) getRedisChanName() string {
	return fmt.Sprintf("bk-cache:del:%v", bc.AppName)
}

type cacheMessage struct {
	Sec  int64
	Nano int
	Key  string
}

// LoadCacheGlobeConfig -
func LoadCacheGlobeConfig() {
	WarnLogMode = viper.GetBool("cache.warn_log")
	DebugLogMode = viper.GetBool("cache.debug_log")
}

// InitRedis will set redis connection to the cache
func (bc *BkCacheStruct) InitRedis(ctx context.Context, rdConn *bkredis.RedisClient, appName string) {
	bc.SetCacheConfig()
	if rdConn != nil {
		bc.RedisFeatured = true
		bc.Redis = rdConn
	} else {
		logger.BkLog.Warn("Init redis but got nil")
		return
	}

	// TODO: need refactor, move all reporter into 1 place
	if bc.SlowCacheReportMode {
		logger.BkLog.Infof("Init slow cache report: %v", bc.SlowCacheReportMode)
		GetProducerFromConf()
		go func() {
			slowCacheReportTicker := time.NewTicker(5 * time.Minute)
			defer slowCacheReportTicker.Stop()
			for {
				select {
				case <-slowCacheReportTicker.C:
					slowGet := bc.slowCacheGetCount
					if slowGet > 0 {
						logger.BkLog.Infof("[Slow cache report] get command counter: %v", slowGet)
						reportCacheGetData, _ := json.Marshal(map[string]int64{"slow_cache_get_counter": slowGet})
						slowGetErr := PublishWithRoutingKey("core_cache_report", "slow_cache_get", reportCacheGetData)
						if slowGetErr == nil {
							atomic.StoreInt64(&bc.slowCacheGetCount, 0)
						}
					}

					slowSet := bc.slowCacheSetCount
					if slowSet > 0 {
						logger.BkLog.Infof("[Slow cache report] set command counter: %v", slowSet)
						reportCacheSetData, _ := json.Marshal(map[string]int64{"slow_cache_set_counter": slowSet})
						slowSetErr := PublishWithRoutingKey("core_cache_report", "slow_cache_set", reportCacheSetData)
						if slowSetErr == nil {
							atomic.StoreInt64(&bc.slowCacheSetCount, 0)
						}
					}

				case <-bc.lastCtx.Done():
					return

				default:
					// always sleep: sleep time should be increased follow by waiting time
					time.Sleep(time.Second * 10)
				}
			}
		}()
	}

	var cancelFunc func()
	bc.lastCtx, cancelFunc = utils.GetContextWithCancelSafe2()
	bc.cancelFuncs = append(bc.cancelFuncs, cancelFunc)

	if bc.cacheSyncEnabled {
		// setup cache flush pub-sub
		logger.BkLog.Debugf("Setup flush local cache")
		bc.AppName = appName
		chanName := bc.getRedisChanName()
		pubsub := bc.Redis.Subscribe(ctx, chanName)

		// Wait for confirmation that subscription is created before publishing anything.
		_, err := pubsub.Receive(ctx)
		if err != nil {
			logger.BkLog.Errorw("Error when subscribe channel", "error", err, "channel", chanName)
			panic(err)
		} else {
			logger.BkLog.Infof("Start listen chan %v for async", chanName)
			// Go channel which receives messages.
			ch := pubsub.Channel()
			inChan := make(chan string)
			outChan := make(chan string)

			bufferChan := utils.NewBufferChan(BufferChanSize)
			go bufferChan.BufferString(inChan, outChan)

			go func() {
				for {
					select {
					case msg := <-ch:
						if msg != nil {
							var data cacheMessage
							jerr := json.Unmarshal([]byte(msg.Payload), &data)
							var k string
							if jerr != nil {
								log("Deleting cache: ", msg.Payload)
								k = msg.Payload
							} else {
								publishTime := time.Unix(data.Sec, int64(data.Nano))
								movingTime := time.Since(publishTime)
								log("Deleting cache: ", data.Key, ", since ", movingTime, " from ", publishTime)
								k = data.Key
								if movingTime > WarnThreshold {
									warning("Warn: take too much time to receive flush key message ", movingTime, ", key: ", k)
								}
							}

							inChan <- k
						}
					case <-bc.lastCtx.Done():

						bufferChan.Close()
						return
					default:
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()

			// clear local cache
			go func() {
				for {
					select {
					case <-bc.lastCtx.Done():
					case k := <-outChan:
						bc.lock.Lock()
						bc.LocalCache.Remove(k)
						bc.lock.Unlock()

						log("Deleted cache: ", k, ", buff len:", bufferChan.Len())
					default:
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()
		}
	}

	//go bc.delLoop(utils.GetContextWithRandomKV(lastCtx))
	go bc.setLoop(utils.GetContextWithRandomKV(bc.lastCtx))
}

// DeleteAdv --
func (bc *BkCacheStruct) DeleteAdv(ctx context.Context, key string, localOnly bool) {
	// Delete local cache
	if bc.LocalFeatured {
		bc.lock.Lock()
		bc.LocalCache.Remove(key)
		bc.lock.Unlock()
	}
	if bc.RedisFeatured && !localOnly {
		if err := bc.Redis.Del(ctx, key).Err(); err != nil {
			logger.BkLog.Errorw("Error when del redis key", "error", err, "key", key)
		}
		if bc.cacheSyncEnabled {
			now := time.Now()
			// flush cache on all node
			data, err := json.Marshal(cacheMessage{
				Sec:  now.Unix(),
				Nano: now.Nanosecond(),
				Key:  key,
			})
			if err != nil {
				logger.BkLog.Infof("Error when serialize cache message: %v", err)
			} else {
				err := bc.Redis.Publish(ctx, bc.getRedisChanName(), string(data)).Err()
				if err != nil {
					logger.BkLog.Errorw("Error when push to redis chan", "error", err, "chan", bc.getRedisChanName())
				}
			}
		}
		atomic.AddUint64(&bc.delCount, 1)
	}
}

// Delete --
func (bc *BkCacheStruct) Delete(ctx context.Context, key string) {
	bc.DeleteAdv(ctx, key, false)
}

// SetCacheAdv set item to redis cache
func (bc *BkCacheStruct) SetCacheAdv(key string, value interface{},
	ttl int, localOnly bool) {
	item := CachedItem{Ct: time.Now().UnixNano(), V: value}
	if ttl <= 1 || ttl > DefaultMaxDayPeriod { // not set or misused ttl field value
		item.LDt = DefaultLocalCacheTimeoutPeriod
		item.RDt = DefaultRemoteCacheTimeoutPeriod
	} else {
		localTTL := ttl / 2
		if ttl < 5 {
			localTTL = ttl
		}
		item.LDt = int64(time.Duration(localTTL) * time.Second)
		item.RDt = int64(time.Duration(ttl) * time.Second)
	}

	valueStr, err := codec.Encode(&item)
	if err != nil {
		logger.BkLog.Errorw("Cannot encode data for key-value pair:", "error", err, "key", key, "value", value)
		return
	}

	atomic.AddUint64(&bc.totalSet, 1)
	if bc.LocalFeatured {
		bc.lock.Lock()
		bc.LocalCache.Add(key, valueStr)
		bc.lock.Unlock()
	}
	if !localOnly && bc.RedisFeatured {
		bc.sets <- &setData{key: key, value: valueStr, ttl: ttl}
	}
}

// SetCache set cache simple version. It use some default params for timeout and cache level
func (bc *BkCacheStruct) SetCache(key string, value interface{}) {
	start := time.Now()
	bc.SetCacheAdv(key, value, DefaultRemoteCacheTimeout, false)
	if time.Since(start) > WarnThreshold {
		atomic.AddInt64(&bc.slowCacheSetCount, 1)
		warning("Warn: take too much time to set cache, ", time.Since(start), ", key:", key)
	}
}

// SetCacheWithTimeout SetCache set cache with timeout
func (bc *BkCacheStruct) SetCacheWithTimeout(key string, value interface{}, timeout int) {
	start := time.Now()
	bc.SetCacheAdv(key, value, timeout, false)
	if time.Since(start) > WarnThreshold {
		atomic.AddInt64(&bc.slowCacheSetCount, 1)
		warning("Warn: take too much time to set cache, ", time.Since(start), ", key:", key)
	}
}

// SetCacheProtoTTL set cache simple version with timeout param
func (bc *BkCacheStruct) SetCacheProtoTTL(key string, value proto.Message, ttl int64) {
	bc.setCacheProtof(key, value, codec.EncodeProto, ttl)
}

// SetCacheProto set cache simple version. It use some default params for timeout and cache level
func (bc *BkCacheStruct) SetCacheProto(key string, value proto.Message) {
	bc.setCacheProtof(key, value, codec.EncodeProto, DefaultRemoteCacheTimeout)
}

func (bc *BkCacheStruct) setCacheProtof(key string, value proto.Message, f codec.EncodeProtoF, ttl int64) {
	start := time.Now()
	d, e := f(value)
	if e != nil {
		return
	}

	bc.SetCacheAdv(key, d, int(ttl), false)
	if time.Since(start) > WarnThreshold {
		atomic.AddInt64(&bc.slowCacheSetCount, 1)
		warning("Warn: take too much time to set cache, ", time.Since(start), ", key:", key)
	}
}

// GetCache get cache by key
//
// Return
//  data in interface type
//  an integer indicate the hit level
//  error
func (bc *BkCacheStruct) GetCache(ctx context.Context, key string) (interface{}, int, error) {
	start := time.Now()
	v, hit, err := bc.GetCacheAdv(ctx, key)
	if time.Since(start) > WarnThreshold {
		warning("Warn: take too much time to get cache ", time.Since(start), ", key:", key, ", hit:", hit, ", err:", err)
	}
	return v, hit, err
}

// GetCacheProto get cache by key
//
// Return
//  data in interface type
//  an integer indicate the hit level
//  error
func (bc *BkCacheStruct) GetCacheProto(ctx context.Context, key string, result proto.Message) (int, error) {
	return bc.getCacheProtof(ctx, key, result, codec.DecodeProto)
}

func (bc *BkCacheStruct) getCacheProtof(ctx context.Context, key string, result proto.Message, f codec.DecodeProtoF) (int, error) {
	start := time.Now()
	v, hit, err := bc.GetCacheAdv(ctx, key)
	if hit <= 0 || err != nil {
		return hit, err
	}
	err = f(v.([]byte), result)

	if time.Since(start) > WarnThreshold {
		warning("Warn: take too much time to get cache ", time.Since(start), ", key:", key, ", hit:", hit, ", err:", err)
	}
	return hit, err
}

// GetCacheAdv get cache by key
//
// Return
//  data in interface type
//  an integer indicate the hit level
//  error
func (bc *BkCacheStruct) GetCacheAdv(ctx context.Context, key string) (interface{}, int, error) {
	start := time.Now()
	defer func() {
		since := time.Since(start)
		atomic.AddUint64(&bc.getTime, uint64(since/time.Millisecond))
		if since > (50 * time.Millisecond) {
			atomic.AddInt64(&bc.slowCacheGetCount, 1)
			logger.BkLog.Info("slow cache: ", key, ", ", since)
		}
	}()
	var tmp interface{}
	var err error
	hit := CacheMiss
	var ok bool
	bc.lock.Lock()
	atomic.AddUint64(&bc.total, 1)
	tmp, ok = bc.LocalCache.Get(key)
	bc.lock.Unlock()
	expired := false
remote:
	if !ok || (expired && hit == CacheHitLocal) {
		if bc.RedisFeatured {
			cmd := bc.Redis.Get(ctx, key)
			tmp, err = cmd.Bytes()
			if err != nil {
				if err != redis.Nil {
					atomic.AddUint64(&bc.err, 1)
					return nil, CacheError, err
				}
				atomic.AddUint64(&bc.miss, 1)
				return nil, CacheMiss, nil
			}
			atomic.AddUint64(&bc.hitRemote, 1)
			hit = CacheHitRemote
		} else {
			if expired {
				return nil, CacheExpired, nil
			}
			atomic.AddUint64(&bc.miss, 1)
			return nil, CacheMiss, nil
		}
	} else {
		atomic.AddUint64(&bc.hitLocal, 1)
		hit = CacheHitLocal
	}

	item := CachedItem{}
	err = codec.DecodeToStruct(tmp.([]byte), &item)
	if err != nil {
		atomic.AddUint64(&bc.err, 1)
		return nil, CacheError, err
	}

	if item.expired(hit) {
		if hit == CacheHitLocal {
			bc.lock.Lock()
			bc.LocalCache.Remove(key)
			bc.lock.Unlock()
			atomic.AddUint64(&bc.expired, 1)
			expired = true
			goto remote
		} else {
			return nil, CacheExpired, nil
		}
	}

	// set back to local cache if needed
	if hit == CacheHitRemote && bc.LocalFeatured {
		maxTime := item.Ct + item.RDt
		item.Ct = time.Now().UnixNano()
		if lowerBound := maxTime - item.Ct; lowerBound < item.LDt {
			item.LDt = lowerBound
		}
		val, err := codec.Encode(&item)
		if err == nil {
			bc.lock.Lock()
			defer bc.lock.Unlock()
			bc.LocalCache.Add(key, val)
		} else {
			logger.BkLog.Errorw("Cannot encode data for key-value pair", "error", err, "key", key, "value", item.V)
			atomic.AddUint64(&bc.err, 1)
		}
	}
	return item.V, hit, nil
}

// Close cache
func (bc *BkCacheStruct) Close() {
	if bc.RedisFeatured {
		for _, f := range bc.cancelFuncs {
			f()
		}
		time.Sleep(time.Second)
		bc.Redis.Close()
	}
	defer close(bc.del)
	defer close(bc.sets)
}

func warning(val ...interface{}) {
	if WarnLogMode {
		logger.BkLog.Warn(val...)
	} else {
		logger.BkLog.Info(val...)
	}
}

func log(val ...interface{}) {
	if DebugLogMode {
		logger.BkLog.Info(val...)
	}
}

func logf(format string, val ...interface{}) {
	if DebugLogMode {
		logger.BkLog.Infof(format, val...)
	}
}
