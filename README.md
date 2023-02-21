# gorm-cache

`gorm-cache` 旨在为gorm v2用户提供一个即插即用的旁路缓存解决方案。本缓存只适用于数据库表单主键时的场景。

## 特性
- 即插即用
- 旁路缓存
- 穿透防护
- 多存储介质（内存/redis）

## 使用说明

```go
import (
    "context"
    "github.com/asjdf/gorm-cache/cache"
    "github.com/redis/go-redis/v9"
)

func main() {
    dsn := "user:pass@tcp(127.0.0.1:3306)/database_name?charset=utf8mb4"
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",    
    })
    
    cache, _ := cache.NewGorm2Cache(&config.CacheConfig{
        CacheLevel:           config.CacheLevelAll,
        CacheStorage:         storage.NewRedisWithClient(redisClient),
        InvalidateWhenUpdate: true, // when you create/update/delete objects, invalidate cache
        CacheTTL:             5000, // 5000 ms
        CacheMaxItemCnt:      50,   // if length of objects retrieved one single time 
                                    // exceeds this number, then don't cache
    })
    // More options in `config.config.go`
    db.Use(cache)    // use gorm plugin
    // cache.AttachToDB(db)

    var users []User
    
    db.Where("value > ?", 123).Find(&users) // search cache not hit, objects cached
    db.Where("value > ?", 123).Find(&users) // search cache hit
    
    db.Where("id IN (?)", []int{1, 2, 3}).Find(&users) // primary key cache not hit, users cached
    db.Where("id IN (?)", []int{1, 3}).Find(&users) // primary key cache hit
}
```

在gorm中主要有5种操作（括号中是gorm中对应函数名）:

1. Query (First/Take/Last/Find/FindInBatches/FirstOrInit/FirstOrCreate/Count/Pluck)
2. Create (Create/CreateInBatches/Save)
3. Delete (Delete)
4. Update (Update/Updates/UpdateColumn/UpdateColumns/Save)
5. Row (Row/Rows/Scan)

本库不支持Row操作的缓存。（WIP）

## 存储介质细节

本库支持使用2种 cache 存储介质：

1. 内存 (ccache/gcache)
2. Redis (所有数据存储在redis中，如果你有多个实例使用本缓存，那么他们不共享redis存储空间)

并且允许多个gorm-cache公用一个存储池，以确保同一数据库的多个gorm实例共享缓存。
