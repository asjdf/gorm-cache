package cache

import (
	"sync"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/util"
	"gorm.io/gorm"
)

func AfterUpdate(cache *Gorm2Cache) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.RowsAffected == 0 {
			return // no rows affected, no need to invalidate cache
		}

		tableName := ""
		if db.Statement.Schema != nil {
			tableName = db.Statement.Schema.Table
		} else {
			tableName = db.Statement.Table
		}
		ctx := db.Statement.Context

		if db.Error == nil && cache.Config.InvalidateWhenUpdate && util.ShouldCache(tableName, cache.Config.Tables) {
			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()

				if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlyPrimary {
					primaryKeys := getPrimaryKeysFromWhereClause(db)
					cache.Logger.CtxInfo(ctx, "[AfterUpdate] parse primary keys = %v", primaryKeys)

					if len(primaryKeys) > 0 {
						cache.Logger.CtxInfo(ctx, "[AfterUpdate] now start to invalidate cache for primary keys: %+v",
							primaryKeys)
						err := cache.BatchInvalidatePrimaryCache(ctx, tableName, primaryKeys)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterUpdate] invalidating primary cache for key %v error: %v",
								primaryKeys, err)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterUpdate] invalidating cache for primary keys: %+v finished.", primaryKeys)
					} else {
						cache.Logger.CtxInfo(ctx, "[AfterUpdate] now start to invalidate all primary cache for table: %s", tableName)
						err := cache.InvalidateAllPrimaryCache(ctx, tableName)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterUpdate] invalidating primary cache for table %s error: %v",
								tableName, err)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterUpdate] invalidating all primary cache for table: %s finished.", tableName)
					}

					// 失效unique键缓存
					// 尝试从WHERE子句中提取unique键
					uniqueKeysMap := getUniqueKeysFromWhereClause(db)
					if len(uniqueKeysMap) > 0 {
						for indexName, uniqueKeys := range uniqueKeysMap {
							if len(uniqueKeys) > 0 {
								cache.Logger.CtxInfo(ctx, "[AfterUpdate] now start to invalidate unique cache for index %s keys: %+v", indexName, uniqueKeys)
								err := cache.BatchInvalidateUniqueCache(ctx, tableName, indexName, uniqueKeys)
								if err != nil {
									cache.Logger.CtxError(ctx, "[AfterUpdate] invalidating unique cache for index %s keys %v error: %v",
										indexName, uniqueKeys, err)
								} else {
									cache.Logger.CtxInfo(ctx, "[AfterUpdate] invalidating unique cache for index %s keys: %+v finished.", indexName, uniqueKeys)
								}
							}
						}
					} else {
						// 如果没有从WHERE子句提取到unique键，失效所有unique键缓存
						if db.Statement.Schema != nil {
							allUniqueIndexes := getAllUniqueIndexes(db.Statement.Schema)
							for indexName := range allUniqueIndexes {
								cache.Logger.CtxInfo(ctx, "[AfterUpdate] now start to invalidate all unique cache for index %s", indexName)
								err := cache.InvalidateAllUniqueCache(ctx, tableName, indexName)
								if err != nil {
									cache.Logger.CtxError(ctx, "[AfterUpdate] invalidating all unique cache for index %s error: %v", indexName, err)
								} else {
									cache.Logger.CtxInfo(ctx, "[AfterUpdate] invalidating all unique cache for index %s finished.", indexName)
								}
							}
						}
					}
				}
			}()

			go func() {
				defer wg.Done()

				if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlySearch {
					cache.Logger.CtxInfo(ctx, "[AfterUpdate] now start to invalidate search cache for table: %s", tableName)
					err := cache.InvalidateSearchCache(ctx, tableName)
					if err != nil {
						cache.Logger.CtxError(ctx, "[AfterUpdate] invalidating search cache for table %s error: %v",
							tableName, err)
						return
					}
					cache.Logger.CtxInfo(ctx, "[AfterUpdate] invalidating search cache for table: %s finished.", tableName)
				}
			}()

			if !cache.Config.AsyncWrite {
				wg.Wait()
			}
		}
	}
}
