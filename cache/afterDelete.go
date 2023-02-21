package cache

import (
	"sync"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/util"
	"gorm.io/gorm"
)

func AfterDelete(cache *Gorm2Cache) func(db *gorm.DB) {
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
					if len(primaryKeys) > 0 {
						cache.Logger.CtxInfo(ctx, "[AfterDelete] now start to invalidate cache for primary keys: %v",
							primaryKeys)
						err := cache.BatchInvalidatePrimaryCache(ctx, tableName, primaryKeys)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterDelete] invalidating cache for primary keys: %v error: %v",
								primaryKeys, err)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterDelete] invalidating cache for primary keys: %v finished.", primaryKeys)
					} else {
						cache.Logger.CtxInfo(ctx, "[AfterDelete] now start to invalidate all primary cache for table: %s", tableName)
						err := cache.InvalidateAllPrimaryCache(ctx, tableName)
						if err != nil {
							cache.Logger.CtxError(ctx, "[AfterDelete] invalidating primary cache for table %s error: %v",
								tableName, err)
							return
						}
						cache.Logger.CtxInfo(ctx, "[AfterDelete] invalidating all primary cache for table: %s finished.", tableName)
					}

				}
			}()

			go func() {
				defer wg.Done()

				if cache.Config.CacheLevel == config.CacheLevelAll || cache.Config.CacheLevel == config.CacheLevelOnlySearch {
					cache.Logger.CtxInfo(ctx, "[AfterDelete] now start to invalidate search cache for table: %s", tableName)
					err := cache.InvalidateSearchCache(ctx, tableName)
					if err != nil {
						cache.Logger.CtxError(ctx, "[AfterDelete] invalidating search cache for table %s error: %v",
							tableName, err)
						return
					}
					cache.Logger.CtxInfo(ctx, "[AfterDelete] invalidating search cache for table: %s finished.", tableName)
				}
			}()

			if !cache.Config.AsyncWrite {
				wg.Wait()
			}
		}
	}
}
