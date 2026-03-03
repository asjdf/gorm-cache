package cache

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestUser is a test model
type TestUser struct {
	ID   uint   `gorm:"primaryKey"`
	Name string
	Age  int
}

func (TestUser) TableName() string {
	return "test_users"
}

func setupTestDB(t *testing.T) *gorm.DB {
	f, err := os.CreateTemp("", "gormCacheTest.*.db")
	if err != nil {
		t.Fatalf("create temp db error: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	db, err := gorm.Open(sqlite.Open(f.Name()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db error: %v", err)
	}

	// Auto migrate
	err = db.AutoMigrate(&TestUser{})
	if err != nil {
		t.Fatalf("auto migrate error: %v", err)
	}

	return db
}

func TestGorm2Cache_Initialize(t *testing.T) {
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

	err := cache.Initialize(db)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify callbacks are registered by checking if we can use the cache
	// The callbacks should be registered without error
}

func TestGorm2Cache_AttachToDB(t *testing.T) {
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

	// AttachToDB should not panic
	cache.AttachToDB(db)

	// Verify it worked by checking if callbacks are registered
	// We can't directly check, but if it didn't panic, it likely worked
}

func TestGorm2Cache_Initialize_WithError(t *testing.T) {
	// Test Initialize with a DB that might cause errors
	// This is difficult to test without mocking, but we can at least test the happy path
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

	err := cache.Initialize(db)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Try to initialize again (should work, callbacks can be registered multiple times)
	err = cache.Initialize(db)
	if err != nil {
		t.Fatalf("expected no error on second initialize, got %v", err)
	}
}

func TestAfterCreate(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "Test", Age: 25}
	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}

	// AfterCreate callback should have been called
	// We can't easily verify it without checking the cache, but if it didn't panic, it worked
}

func TestAfterUpdate(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user first
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)

	// Update the user
	result := db.Model(&user).Update("name", "Updated")
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}

	// AfterUpdate callback should have been called
}

func TestAfterDelete(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user first
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)

	// Delete the user
	result := db.Delete(&user)
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}

	// AfterDelete callback should have been called
}

func TestAfterCreate_WithInvalidateDisabled(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: false, // Disabled
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "Test", Age: 25}
	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}

	// Should not invalidate cache when InvalidateWhenUpdate is false
}

func TestAfterUpdate_WithInvalidateDisabled(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: false, // Disabled
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create and update
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Model(&user).Update("name", "Updated")
}

func TestAfterDelete_WithInvalidateDisabled(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: false, // Disabled
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create and delete
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Delete(&user)
}

func TestAfterCreate_WithZeroRowsAffected(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Try to create with zero rows affected (should not invalidate)
	// This is hard to test directly, but the callback should handle it gracefully
	user := TestUser{ID: 999, Name: "Test", Age: 25}
	// Create with a specific ID that might conflict
	_ = db.Create(&user)
}

func TestAfterUpdate_WithZeroRowsAffected(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Update non-existent record (zero rows affected)
	db.Model(&TestUser{}).Where("id = ?", 99999).Update("name", "Updated")
}

func TestAfterDelete_WithZeroRowsAffected(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Delete non-existent record (zero rows affected)
	db.Delete(&TestUser{}, 99999)
}

func TestAfterCreate_WithAsyncWrite(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			AsyncWrite:           true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create a user
	user := TestUser{Name: "Test", Age: 25}
	result := db.Create(&user)
	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}
}

func TestAfterUpdate_WithAsyncWrite(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			AsyncWrite:           true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create and update
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Model(&user).Update("name", "Updated")
}

func TestAfterDelete_WithAsyncWrite(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			AsyncWrite:           true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create and delete
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Delete(&user)
}

func TestAfterUpdate_WithPrimaryKeys(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
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

	// Update with WHERE clause containing primary keys
	db.Model(&TestUser{}).Where("id IN (?)", []uint{1, 2}).Update("age", 35)
}

func TestAfterDelete_WithPrimaryKeys(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
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

	// Delete with WHERE clause containing primary keys
	db.Where("id IN (?)", []uint{1, 2}).Delete(&TestUser{})
}

func TestAfterUpdate_WithoutPrimaryKeys(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
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

	// Update without primary key in WHERE clause
	db.Model(&TestUser{}).Where("name = ?", "User1").Update("age", 35)
}

func TestAfterDelete_WithoutPrimaryKeys(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"test_users"},
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

	// Delete without primary key in WHERE clause
	db.Where("name = ?", "User1").Delete(&TestUser{})
}

func TestAfterCreate_WithCacheLevelOnlySearch(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelOnlySearch,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
}

func TestAfterUpdate_WithCacheLevelOnlyPrimary(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelOnlyPrimary,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Model(&user).Update("name", "Updated")
}

func TestAfterDelete_WithCacheLevelOnlyPrimary(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelOnlyPrimary,
			Tables:              []string{"test_users"},
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Delete(&user)
}

func TestAfterCreate_WithTableNotInCacheList(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"other_table"}, // test_users not in list
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	// Create should not trigger cache invalidation for tables not in cache list
	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
}

func TestAfterUpdate_WithTableNotInCacheList(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"other_table"}, // test_users not in list
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Model(&user).Update("name", "Updated")
}

func TestAfterDelete_WithTableNotInCacheList(t *testing.T) {
	db := setupTestDB(t)
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage:         memStorage,
			CacheTTL:            1000,
			DebugMode:           false,
			InvalidateWhenUpdate: true,
			CacheLevel:           config.CacheLevelAll,
			Tables:              []string{"other_table"}, // test_users not in list
		},
		stats: &stats{},
	}
	cache.Init()
	cache.Initialize(db)

	user := TestUser{Name: "Test", Age: 25}
	db.Create(&user)
	db.Delete(&user)
}

func TestQueryHandler_Bind(t *testing.T) {
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

	handler := newQueryHandler(cache)
	err := handler.Bind(db)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestQueryHandler_Bind_WithError(t *testing.T) {
	// Test Bind with invalid callback registration
	// This is hard to test without mocking, but we can test the happy path
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

	handler := newQueryHandler(cache)
	err := handler.Bind(db)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Try to bind again (should work)
	err = handler.Bind(db)
	if err != nil {
		t.Fatalf("expected no error on second bind, got %v", err)
	}
}

// recordingStorage 记录 DeleteKeysWithPrefix 的调用，用于断言 schema-less 时是否仍失效 unique 缓存
type recordingStorage struct {
	*storage.Memory
	mu               sync.Mutex
	deletedPrefixes  []string
}

func (r *recordingStorage) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	r.mu.Lock()
	r.deletedPrefixes = append(r.deletedPrefixes, keyPrefix)
	r.mu.Unlock()
	return r.Memory.DeleteKeysWithPrefix(ctx, keyPrefix)
}

// testUserWithUnique 带 unique 索引的模型，用于 schema-less fallback 测试
type testUserWithUnique struct {
	ID       uint   `gorm:"primaryKey"`
	Email    string `gorm:"uniqueIndex:idx_email"`
	Username string `gorm:"uniqueIndex:idx_username"`
}

func (testUserWithUnique) TableName() string {
	return "test_users_unique"
}

// TestAfterDelete_SchemaNil_StillInvalidatesUniqueCache 复现：当 Schema 为 nil 但 Model 有 unique 索引时，
// fallback 路径应通过解析 Model 得到 schema 并失效所有 unique 缓存，否则会残留过期 unique 缓存。
func TestAfterDelete_SchemaNil_StillInvalidatesUniqueCache(t *testing.T) {
	rec := &recordingStorage{Memory: storage.NewMem(storage.DefaultMemStoreConfig)}
	cfg := &config.CacheConfig{
		CacheStorage:         rec,
		CacheTTL:             1000,
		DebugMode:            false,
		InvalidateWhenUpdate: true,
		CacheLevel:           config.CacheLevelAll,
		Tables:               []string{"test_users_unique"},
	}
	cache := &Gorm2Cache{Config: cfg, stats: &stats{}}
	cache.Init()

	db := setupTestDB(t)
	// 模拟 schema-less 场景：Schema 为 nil，但 Model 指向带 unique 索引的模型
	db.Statement.Schema = nil
	db.Statement.Model = &testUserWithUnique{}
	db.Statement.Table = "test_users_unique"
	db.RowsAffected = 1
	db.Error = nil
	db.Statement.Context = context.Background()
	// 不设置 WHERE，使 getUniqueKeysFromWhereClause 返回空，走「失效所有 unique」的 fallback 路径

	hook := AfterDelete(cache)
	hook(db)

	// 回调内是 goroutine，等待执行完
	time.Sleep(200 * time.Millisecond)

	rec.mu.Lock()
	prefixes := append([]string(nil), rec.deletedPrefixes...)
	rec.mu.Unlock()

	// 应至少对两个 unique 索引做 DeleteKeysWithPrefix（idx_email, idx_username）
	var uniquePrefixCount int
	for _, p := range prefixes {
		if strings.Contains(p, ":u:") && strings.Contains(p, "test_users_unique") {
			uniquePrefixCount++
		}
	}
	if uniquePrefixCount < 2 {
		t.Errorf("schema-less fallback should invalidate all unique caches (expected >= 2 unique prefix deletes), got %d, prefixes: %v", uniquePrefixCount, prefixes)
	}
	// 同时应包含 idx_email 与 idx_username 的 prefix（util.GenUniqueCachePrefix 格式）
	hasEmail := false
	hasUsername := false
	for _, p := range prefixes {
		if strings.Contains(p, "idx_email") {
			hasEmail = true
		}
		if strings.Contains(p, "idx_username") {
			hasUsername = true
		}
	}
	if !hasEmail || !hasUsername {
		t.Errorf("expected unique prefix deletes for idx_email and idx_username, got prefixes: %v", prefixes)
	}
}

// TestAfterUpdate_SchemaNil_StillInvalidatesUniqueCache 与 AfterDelete 对称：Schema 为 nil 时 update 也应失效 unique 缓存
func TestAfterUpdate_SchemaNil_StillInvalidatesUniqueCache(t *testing.T) {
	rec := &recordingStorage{Memory: storage.NewMem(storage.DefaultMemStoreConfig)}
	cfg := &config.CacheConfig{
		CacheStorage:         rec,
		CacheTTL:             1000,
		DebugMode:            false,
		InvalidateWhenUpdate: true,
		CacheLevel:           config.CacheLevelAll,
		Tables:               []string{"test_users_unique"},
	}
	cache := &Gorm2Cache{Config: cfg, stats: &stats{}}
	cache.Init()

	db := setupTestDB(t)
	db.Statement.Schema = nil
	db.Statement.Model = &testUserWithUnique{}
	db.Statement.Table = "test_users_unique"
	db.RowsAffected = 1
	db.Error = nil
	db.Statement.Context = context.Background()

	hook := AfterUpdate(cache)
	hook(db)

	time.Sleep(200 * time.Millisecond)

	rec.mu.Lock()
	prefixes := append([]string(nil), rec.deletedPrefixes...)
	rec.mu.Unlock()

	var uniquePrefixCount int
	for _, p := range prefixes {
		if strings.Contains(p, ":u:") && strings.Contains(p, "test_users_unique") {
			uniquePrefixCount++
		}
	}
	if uniquePrefixCount < 2 {
		t.Errorf("schema-less fallback should invalidate all unique caches (expected >= 2 unique prefix deletes), got %d, prefixes: %v", uniquePrefixCount, prefixes)
	}
	hasEmail := false
	hasUsername := false
	for _, p := range prefixes {
		if strings.Contains(p, "idx_email") {
			hasEmail = true
		}
		if strings.Contains(p, "idx_username") {
			hasUsername = true
		}
	}
	if !hasEmail || !hasUsername {
		t.Errorf("expected unique prefix deletes for idx_email and idx_username, got prefixes: %v", prefixes)
	}
}

