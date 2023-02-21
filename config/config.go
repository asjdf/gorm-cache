package config

import (
	"github.com/asjdf/gorm-cache/storage"
	"github.com/asjdf/gorm-cache/util"
)

type CacheConfig struct {
	// CacheLevel there are 2 types of cache and 4 kinds of cache option
	CacheLevel CacheLevel

	// CacheStorage choose proper storage medium
	CacheStorage storage.DataStorage

	// Tables only cache data within given data tables (cache all if empty)
	Tables []string

	// InvalidateWhenUpdate
	// if user update/delete/create something in DB, we invalidate all cached data to ensure consistency,
	// else we do nothing to outdated cache.
	InvalidateWhenUpdate bool

	// AsyncWrite if true, then we will write cache in async mode
	AsyncWrite bool

	// CacheTTL cache ttl in ms, where 0 represents forever
	CacheTTL int64

	// CacheMaxItemCnt for given query, if objects retrieved are more than this cnt,
	// then we choose not to cache for this query. 0 represents caching all queries.
	CacheMaxItemCnt int64

	// DisableCachePenetration if true, then we will not cache nil result
	DisableCachePenetrationProtect bool

	// DebugMode indicate if we're in debug mode (will print access log)
	DebugMode bool

	// DebugLogger
	DebugLogger util.LoggerInterface
}

type CacheLevel int

const (
	CacheLevelOff         CacheLevel = 0
	CacheLevelOnlyPrimary CacheLevel = 1
	CacheLevelOnlySearch  CacheLevel = 2
	CacheLevelAll         CacheLevel = 3
)
