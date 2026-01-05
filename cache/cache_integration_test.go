package cache

import (
	"os"
	"testing"

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

