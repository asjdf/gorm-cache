package cache

import (
	"errors"
	"fmt"
	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
	"github.com/asjdf/gorm-cache/util"
	"github.com/hashicorp/go-multierror"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// singleFlight 流程设计
// 根据key lock住，等待结果。query before之前，会先判断是否有key，如果有，就等待结果，如果没有，就执行query before，然后执行query，然后把结果放到key里面，然后unlock，然后返回结果。
// 等待完成后 进行一手返回 然后err设置为err.singleflightHit，afterQuery结束的时候进行一手检查

func newQueryHandler(c *Gorm2Cache) *queryHandler {
	return &queryHandler{cache: c}
}

type queryHandler struct {
	cache        *Gorm2Cache
	singleFlight Group
}

func (h *queryHandler) Bind(db *gorm.DB) error {
	err := db.Callback().Query().Before("gorm:query").Register("gorm:cache:before_query", h.BeforeQuery())
	if err != nil {
		return err
	}
	err = db.Callback().Query().After("gorm:query").Register("gorm:cache:after_query", h.AfterQuery())
	if err != nil {
		return err
	}
	return nil
}

func (h *queryHandler) BeforeQuery() func(db *gorm.DB) {
	cache := h.cache
	return func(db *gorm.DB) {
		callbacks.BuildQuerySQL(db)
		tableName := ""
		if db.Statement.Schema != nil {
			tableName = db.Statement.Schema.Table
		} else {
			tableName = db.Statement.Table
		}
		ctx := db.Statement.Context

		sql := db.Statement.SQL.String()
		db.InstanceSet("gorm:cache:sql", sql)
		db.InstanceSet("gorm:cache:vars", db.Statement.Vars)

		if util.ShouldCache(tableName, cache.Config.Tables) {
			hit := false
			defer func() {
				if hit {
					cache.IncrHitCount()
				} else {
					cache.IncrMissCount()
				}
			}()

			// singleFlight Check
			singleFlightKey := util.GenSingleFlightKey(tableName, sql, db.Statement.Vars...)
			h.singleFlight.mu.Lock()
			if h.singleFlight.m == nil {
				h.singleFlight.m = make(map[string]*call)
			}
			if c, ok := h.singleFlight.m[singleFlightKey]; ok {
				c.dups++
				h.singleFlight.mu.Unlock()
				c.wg.Wait()

				// 临时糊一个拷贝在这里 性能可能并不是那么好
				d, err := json.Marshal(c.dest)
				if err != nil {
					_ = db.AddError(err)
					return
				}
				err = json.Unmarshal(d, db.Statement.Dest)
				if err != nil {
					_ = db.AddError(err)
					return
				}
				hit = true
				db.RowsAffected = c.rowsAffected
				db.Error = multierror.Append(util.SingleFlightHit) // 为保证后续流程不走，必须设一个error
				if c.err != nil {
					db.Error = multierror.Append(db.Error, c.err)
				}
				h.cache.Logger.CtxInfo(ctx, "[BeforeQuery] single flight hit for key %v", singleFlightKey)
				return
			}
			c := &call{key: singleFlightKey}
			c.wg.Add(1)
			h.singleFlight.m[singleFlightKey] = c
			h.singleFlight.mu.Unlock()
			db.InstanceSet("gorm:cache:query:single_flight_call", c)

			tryPrimaryCache := func() (hitted bool) {
				primaryKeys := getPrimaryKeysFromWhereClause(db)
				cache.Logger.CtxInfo(ctx, "[BeforeQuery] parse primary keys = %v", primaryKeys)

				if len(primaryKeys) == 0 {
					return
				}

				// if (IN primaryKeys)/(Eq primaryKey) are the only clauses
				hasOtherClauseInWhere := hasOtherClauseExceptPrimaryField(db)
				if hasOtherClauseInWhere {
					// if query has other clauses, it can only query the database
					return
				}

				// primary cache hit
				cacheValues, err := cache.BatchGetPrimaryCache(ctx, tableName, primaryKeys)
				if err != nil {
					cache.Logger.CtxError(ctx, "[BeforeQuery] get primary cache value for key %v error: %v", primaryKeys, err)
					db.Error = nil
					return
				}
				if len(cacheValues) != len(primaryKeys) {
					db.Error = nil
					return
				}
				finalValue := ""

				destKind := reflect.Indirect(reflect.ValueOf(db.Statement.Dest)).Kind()
				if destKind == reflect.Struct && len(cacheValues) == 1 {
					finalValue = cacheValues[0]
				} else if (destKind == reflect.Array || destKind == reflect.Slice) && len(cacheValues) >= 1 {
					finalValue = "[" + strings.Join(cacheValues, ",") + "]"
				}
				if len(finalValue) == 0 {
					cache.Logger.CtxError(ctx, "[BeforeQuery] length of cache values and dest not matched")
					db.Error = util.ErrCacheUnmarshal
					return
				}

				err = json.Unmarshal([]byte(finalValue), db.Statement.Dest)
				if err != nil {
					cache.Logger.CtxError(ctx, "[BeforeQuery] unmarshal final value error: %v", err)
					db.Error = util.ErrCacheUnmarshal
					return
				}
				db.Error = util.PrimaryCacheHit
				hitted = true
				return
			}

			trySearchCache := func() (hitted bool) {
				// search cache hit
				cacheValue, err := cache.GetSearchCache(ctx, tableName, sql, db.Statement.Vars...)
				if err != nil {
					if !errors.Is(err, storage.ErrCacheNotFound) {
						cache.Logger.CtxError(ctx, "[BeforeQuery] get cache value for sql %s error: %v", sql, err)
					}
					db.Error = nil
					return
				}
				cache.Logger.CtxInfo(ctx, "[BeforeQuery] get value: %s", cacheValue)
				if cacheValue == "recordNotFound" { // 应对缓存穿透
					db.Error = util.RecordNotFoundCacheHit
					hitted = true
					return
				}
				rowsAffectedPos := strings.Index(cacheValue, "|")
				db.RowsAffected, err = strconv.ParseInt(cacheValue[:rowsAffectedPos], 10, 64)
				if err != nil {
					cache.Logger.CtxError(ctx, "[BeforeQuery] unmarshal rows affected cache error: %v", err)
					db.Error = nil
					return
				}
				err = json.Unmarshal([]byte(cacheValue[rowsAffectedPos+1:]), db.Statement.Dest)
				if err != nil {
					cache.Logger.CtxError(ctx, "[BeforeQuery] unmarshal search cache error: %v", err)
					db.Error = nil
					return
				}
				db.Error = util.SearchCacheHit
				hitted = true
				return
			}

			if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlyPrimary {
				if tryPrimaryCache() {
					hit = true
					return
				}
			}
			if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlySearch {
				if !hit && trySearchCache() {
					hit = true
				}
			}
		}
	}
}

func (h *queryHandler) AfterQuery() func(db *gorm.DB) {
	cache := h.cache
	return func(db *gorm.DB) {
		func() {
			tableName := ""
			if db.Statement.Schema != nil {
				tableName = db.Statement.Schema.Table
			} else {
				tableName = db.Statement.Table
			}
			ctx := db.Statement.Context
			sqlObj, _ := db.InstanceGet("gorm:cache:sql")
			sql := sqlObj.(string)
			varObj, _ := db.InstanceGet("gorm:cache:vars")
			vars := varObj.([]interface{})

			if !util.ShouldCache(tableName, cache.Config.Tables) {
				return
			}

			if db.Error == nil {
				// error is nil -> cache not hit, we cache newly retrieved data
				primaryKeys, objects := getObjectsAfterLoad(db)

				var wg sync.WaitGroup
				wg.Add(2)

				go func() {
					defer wg.Done()

					if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlySearch {
						// cache search data
						if int64(len(objects)) > cache.Config.CacheMaxItemCnt {
							return
						}

						cache.Logger.CtxInfo(ctx, "[AfterQuery] start to set search cache for sql: %s", sql)
						cacheBytes, err := json.Marshal(db.Statement.Dest)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterQuery] cannot marshal cache for sql: %s, not cached", sql)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterQuery] set cache: %v", string(cacheBytes))
						err = cache.SetSearchCache(ctx, fmt.Sprintf("%d|", db.RowsAffected)+string(cacheBytes), tableName, sql, vars...)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterQuery] set search cache for sql: %s error: %v", sql, err)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterQuery] sql %s cached", sql)
					}
				}()

				go func() {
					defer wg.Done()

					if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlyPrimary {
						// cache primary cache data
						if len(primaryKeys) != len(objects) {
							return
						}
						if int64(len(objects)) > cache.Config.CacheMaxItemCnt {
							cache.Logger.CtxInfo(ctx, "[AfterQuery] objects length is more than max item count, not cached")
							return
						}
						kvs := make([]util.Kv, 0, len(objects))
						for i := 0; i < len(objects); i++ {
							jsonStr, err := json.Marshal(objects[i])
							if err != nil {
								cache.Logger.CtxError(ctx, "[AfterQuery] object %v cannot marshal, not cached", objects[i])
								continue
							}
							kvs = append(kvs, util.Kv{
								Key:   primaryKeys[i],
								Value: string(jsonStr),
							})
						}
						cache.Logger.CtxInfo(ctx, "[AfterQuery] start to set primary cache for kvs: %+v", kvs)
						err := cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterQuery] batch set primary key cache for key %v error: %v",
								primaryKeys, err)
						}
					}
				}()
				if !cache.Config.AsyncWrite {
					wg.Wait()
				}
				return
			}

			// 应对缓存穿透 未来可能考虑使用其他过滤器实现：如布隆过滤器
			if db.Error == gorm.ErrRecordNotFound && !cache.Config.DisableCachePenetrationProtect {
				cache.Logger.CtxInfo(ctx, "[AfterQuery] set cache: %v", "recordNotFound")
				err := cache.SetSearchCache(ctx, "recordNotFound", tableName, sql, vars...)
				if err != nil {
					cache.Logger.CtxError(ctx, "[AfterQuery] set search cache for sql: %s error: %v", sql, err)
					return
				}
				cache.Logger.CtxInfo(ctx, "[AfterQuery] sql %s cached", sql)
				return
			}
		}()
		// 之所以将上面的部分包在一个匿名函数中是为了方便
		// 上面的cache完成后直接传播给其他等待中的goroutine
		// 上面只处理非singleflight且无错误或记录不存在的情况
		h.fillCallAfterQuery(db)

		// 下面处理命中了缓存的情况
		// 有以下几种err是专门用来传状态的：正常的cachehit 这种情况不存在error
		// RecordNotFoundCacheHit 这种情况只会在notfound之后出现
		// SingleFlightHit 这种情况下error中除了SingleFlightHit还可能会存在其他error来自gorm的error
		// 且遇到任何一种hit我们都可以认为是命中了缓存 同时只可能命中至多两个hit（single+其他
		if merr, ok := db.Error.(*multierror.Error); ok {
			errs := merr.WrappedErrors()
			if errors.Is(errs[0], util.SingleFlightHit) {
				if len(errs) > 1 {
					db.Error = errs[1]
				} else {
					db.Error = nil
				}
			}
		}

		switch db.Error {
		case util.RecordNotFoundCacheHit:
			db.Error = gorm.ErrRecordNotFound
		case util.SearchCacheHit, util.PrimaryCacheHit:
			db.Error = nil
		}
	}
}

func (h *queryHandler) fillCallAfterQuery(db *gorm.DB) {
	if singleFlightCallObj, exist := db.InstanceGet("gorm:cache:query:single_flight_call"); exist {
		c := singleFlightCallObj.(*call)
		c.dest = db.Statement.Dest
		c.rowsAffected = db.RowsAffected
		c.err = db.Error
		c.wg.Done()

		h.singleFlight.mu.Lock()
		if !c.forgotten {
			delete(h.singleFlight.m, c.key)
		}
		h.singleFlight.mu.Unlock()
	}
}
