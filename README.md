# gorm-cache

`gorm-cache` 旨在为 gorm v2 用户提供一个即插即用的旁路缓存解决方案。本缓存只适用于数据库表单主键时的场景。

## 特性

-   即插即用
-   旁路缓存
-   穿透防护
-   击穿防护
-   多存储介质（内存/redis）

## 使用说明

```go
import (
    "context"
    "github.com/asjdf/gorm-cache/cache"
    "github.com/asjdf/gorm-cache/storage"
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
        CacheStorage:         storage.NewRedis(&storage.RedisStoreConfig{Client: redisClient}),
        InvalidateWhenUpdate: true, // when you create/update/delete objects, invalidate cache
        CacheTTL:             5000, // 5000 ms
        CacheMaxItemCnt:      50,   // if length of objects retrieved one single time
                                    // exceeds this number, then don't cache
    })
    // More options in `config/config.go`
    db.Use(cache)    // use gorm plugin
    // cache.AttachToDB(db)

    var users []User

    db.Where("value > ?", 123).Find(&users) // search cache not hit, objects cached
    db.Where("value > ?", 123).Find(&users) // search cache hit

    db.Where("id IN (?)", []int{1, 2, 3}).Find(&users) // primary key cache not hit, users cached
    db.Where("id IN (?)", []int{1, 3}).Find(&users) // primary key cache hit
}
```

在 gorm 中主要有 5 种操作（括号中是 gorm 中对应函数名）:

1. Query (First/Take/Last/Find/FindInBatches/FirstOrInit/FirstOrCreate/Count/Pluck)
2. Create (Create/CreateInBatches/Save)
3. Delete (Delete)
4. Update (Update/Updates/UpdateColumn/UpdateColumns/Save)
5. Row (Row/Rows/Scan)

本库不支持 Row 操作的缓存。（WIP）

## 存储介质细节

本库支持使用 2 种 cache 存储介质：

1. 内存 (ccache/gcache)
2. Redis (所有数据存储在 redis 中，如果你有多个实例使用本缓存，那么他们不共享 redis 存储空间)

并且允许多个 gorm-cache 公用一个存储池，以确保同一数据库的多个 gorm 实例共享缓存。

## 测试覆盖率

项目总体测试覆盖率: **85.2%**

### 各包覆盖率

| 包名      | 覆盖率 |
| --------- | ------ |
| `util`    | 100.0% |
| `storage` | 90.8%  |
| `cache`   | 82.1%  |
| `test`    | 100.0% |

### 运行测试

```bash
# 运行所有测试
go test -timeout 60s ./...

# 查看覆盖率
go test -timeout 60s -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 生成 HTML 覆盖率报告
go tool cover -html=coverage.out -o coverage.html
```
