package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
)

func TestGetPrimaryKeysFromWhereClause(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Test with WHERE clause containing primary key
	var users []TestUser
	db.Where("id = ?", 1).Find(&users)
	// The callback should have been called, but we can't directly test getPrimaryKeysFromWhereClause
	// without accessing internal state. This test at least ensures the code path is executed.
}

func TestGetPrimaryKeysFromWhereClause_WithIN(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create some users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
		{Name: "User3", Age: 35},
	}
	db.Create(&users)

	// Query with IN clause on primary key
	var result []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result)
	// This should trigger getPrimaryKeysFromWhereClause with IN expression
}

func TestGetPrimaryKeysFromWhereClause_WithEq(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Query with Eq clause on primary key
	var result TestUser
	db.Where("id = ?", 1).First(&result)
	// This should trigger getPrimaryKeysFromWhereClause with Eq expression
}

func TestGetPrimaryKeysFromWhereClause_WithExpr(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// Query with Expr clause
	var result []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result)
	// This should trigger getPrimaryKeysFromWhereClause with Expr
}

func TestGetPrimaryKeysFromExpr(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// Query that will use Expr parsing
	var result []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result)
	// This should trigger getPrimaryKeysFromExpr
}

func TestGetObjectsAfterLoad(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// Query that will trigger AfterQuery and getObjectsAfterLoad
	var result []TestUser
	db.Find(&result)
	// AfterQuery should call getObjectsAfterLoad to extract primary keys and objects
}

func TestGetObjectsAfterLoad_WithStruct(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Query single struct
	var result TestUser
	db.First(&result)
	// AfterQuery should call getObjectsAfterLoad with struct dest
}

func TestGetObjectsAfterLoad_WithSlice(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// Query slice
	var result []TestUser
	db.Find(&result)
	// AfterQuery should call getObjectsAfterLoad with slice dest
}

func TestBeforeQuery_WithSearchCache(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// First query - should cache
	var result1 []TestUser
	db.Where("name = ?", "User1").Find(&result1)

	// Second query - should hit cache
	var result2 []TestUser
	db.Where("name = ?", "User1").Find(&result2)
	// BeforeQuery should check cache and return cached result
}

func TestBeforeQuery_WithPrimaryCache(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// First query - should cache
	var result1 []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result1)

	// Second query - should hit primary cache
	var result2 []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result2)
	// BeforeQuery should check primary cache and return cached result
}

func TestAfterQuery_WithSearchCache(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Query - AfterQuery should cache the result
	var result []TestUser
	db.Where("name = ?", "User1").Find(&result)
	// AfterQuery should set search cache
}

func TestAfterQuery_WithPrimaryCache(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create users
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
	}
	db.Create(&users)

	// Query - AfterQuery should cache primary keys
	var result []TestUser
	db.Where("id IN (?)", []uint{1, 2}).Find(&result)
	// AfterQuery should set primary cache
}

func TestAfterQuery_WithRecordNotFound(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:                memStorage,
			CacheTTL:                    1000,
			DebugMode:                   false,
			DisableCachePenetrationProtect: false, // Enable protection
			CacheLevel:                  config.CacheLevelAll,
			Tables:                     []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Query non-existent record - should cache "recordNotFound"
	var result TestUser
	db.Where("id = ?", 99999).First(&result)
	// AfterQuery should cache "recordNotFound" to prevent cache penetration
}

func TestAfterQuery_WithRecordNotFound_Disabled(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:                memStorage,
			CacheTTL:                    1000,
			DebugMode:                   false,
			DisableCachePenetrationProtect: true, // Disable protection
			CacheLevel:                  config.CacheLevelAll,
			Tables:                     []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Query non-existent record - should not cache
	var result TestUser
	db.Where("id = ?", 99999).First(&result)
	// AfterQuery should not cache when protection is disabled
}

func TestAfterQuery_WithMaxItemCnt(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:   memStorage,
			CacheTTL:       1000,
			DebugMode:      false,
			CacheMaxItemCnt: 2, // Max 2 items
			CacheLevel:     config.CacheLevelAll,
			Tables:         []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create more than max items
	users := []TestUser{
		{Name: "User1", Age: 25},
		{Name: "User2", Age: 30},
		{Name: "User3", Age: 35},
	}
	db.Create(&users)

	// Query - should not cache because exceeds max item count
	var result []TestUser
	db.Find(&result)
	// AfterQuery should not cache when item count exceeds limit
}

func TestAfterQuery_WithSingleFlight(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Concurrent queries with same SQL - should use singleflight
	done := make(chan bool, 2)
	for i := 0; i < 2; i++ {
		go func() {
			var result []TestUser
			db.Where("name = ?", "User1").Find(&result)
			done <- true
		}()
	}
	<-done
	<-done
	// Singleflight should prevent duplicate queries
}

// TestSingleFlight_ContextCancel_DoesNotAffectWaiters 复现并防护「一人取消，全家报错」：
// leader 使用已取消的 context 抢到执行权时，查库/缓存应用独立 background context，
// 否则 leader 的 cancel 会连累所有等待同一 key 的请求拿到 context.Canceled。
func TestSingleFlight_ContextCancel_DoesNotAffectWaiters(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:    false,
			CacheLevel:   config.CacheLevelAll,
			Tables:       []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "User1", Age: 25}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 模拟「急性子」：leader 用极短超时 context 先抢到 singleflight，查库过程中其 ctx 已过期
	singleFlightLeaderDelayForTest = 50 * time.Millisecond
	defer func() { singleFlightLeaderDelayForTest = 0 }()

	ctxLeader, cancelLeader := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancelLeader()

	var resultLeader, resultWaiter []TestUser
	var errLeader, errWaiter error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		tx := db.WithContext(ctxLeader)
		errLeader = tx.Where("name = ?", "User1").Find(&resultLeader).Error
	}()
	time.Sleep(5 * time.Millisecond) // leader 已抢到锁并在 delay 中，waiter 将等待同一 call
	go func() {
		defer wg.Done()
		tx := db.WithContext(context.Background())
		errWaiter = tx.Where("name = ?", "User1").Find(&resultWaiter).Error
	}()
	wg.Wait()

	// 正确行为：waiter 的 context 从未被取消/超时，不应拿到 leader 的 context 错误（一人取消/超时，全家报错）
	if errWaiter == nil {
		if len(resultWaiter) != 1 || resultWaiter[0].Name != "User1" {
			t.Errorf("waiter should get one record: got %d, err=%v", len(resultWaiter), errWaiter)
		}
	} else {
		if errors.Is(errWaiter, context.Canceled) || errors.Is(errWaiter, context.DeadlineExceeded) {
			t.Errorf("一人取消全家报错：waiter 不应因 leader 的 context 取消/超时而得到相同错误, err=%v", errWaiter)
		}
	}
	// leader 自己拿到 cancel 或超时是可以接受的，不强制断言
	_ = errLeader
	_ = resultLeader
}

func TestAfterQuery_WithAsyncWrite(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			AsyncWrite:  true,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Query - should cache asynchronously
	var result []TestUser
	db.Where("name = ?", "User1").Find(&result)
	// AfterQuery should cache in async mode
}

func TestFillCallAfterQuery(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:     1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "User1", Age: 25}
	db.Create(&user)

	// Query that triggers singleflight
	var result []TestUser
	db.Where("name = ?", "User1").Find(&result)
	// fillCallAfterQuery should be called in AfterQuery
}

func TestGetPrimaryKeysFromWhereClause_NoSchema(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Query without schema (using Table)
	var result []TestUser
	db.Table("test_users").Where("id = ?", 1).Find(&result)
	// Should handle case where Schema is nil
}

func TestGetPrimaryKeysFromWhereClause_NoPrimaryKey(t *testing.T) {
	// Create a model without primary key
	type NoPKModel struct {
		Name string
		Age  int
	}

	db := setupTestDB(t)
	db.Exec("CREATE TABLE IF NOT EXISTS no_pk_models (name TEXT, age INTEGER)")

	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"no_pk_models"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Query table without primary key
	var result []NoPKModel
	db.Table("no_pk_models").Where("name = ?", "Test").Find(&result)
	// Should handle case where no primary key exists
}

func TestGetPrimaryKeysFromWhereClause_NoWhereClause(t *testing.T) {
	db := setupTestDB(t)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
			CacheTTL:    1000,
			DebugMode:   false,
			CacheLevel:  config.CacheLevelAll,
			Tables:      []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Query without WHERE clause
	var result []TestUser
	db.Find(&result)
	// Should handle case where no WHERE clause exists
}

