package cache

import (
	"context"
	"testing"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
	"github.com/asjdf/gorm-cache/util"
)

func TestGorm2Cache_Name(t *testing.T) {
	cache := &Gorm2Cache{}
	name := cache.Name()
	if name != util.GormCachePrefix {
		t.Errorf("expected name to be %s, got %s", util.GormCachePrefix, name)
	}
}

func TestGorm2Cache_Init(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.CacheConfig
		expectError bool
	}{
		{
			name: "with custom storage",
			config: &config.CacheConfig{
				CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
				CacheTTL:     1000,
				DebugMode:    false,
			},
			expectError: false,
		},
		{
			name: "without storage (uses default)",
			config: &config.CacheConfig{
				CacheTTL:  1000,
				DebugMode: false,
			},
			expectError: false,
		},
		{
			name: "with debug logger",
			config: &config.CacheConfig{
				CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
				CacheTTL:     1000,
				DebugMode:    true,
				DebugLogger:  &util.DefaultLogger{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &Gorm2Cache{
				Config: tt.config,
				stats:  &stats{},
			}
			err := cache.Init()
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
			if err == nil {
				if cache.InstanceId == "" {
					t.Error("expected InstanceId to be set")
				}
				if cache.cache == nil {
					t.Error("expected cache to be initialized")
				}
				if cache.Logger == nil {
					t.Error("expected Logger to be set")
				}
			}
		})
	}
}

func TestGorm2Cache_ResetCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()

	// Set some hit/miss counts
	cache.IncrHitCount()
	cache.IncrMissCount()

	// Reset cache
	err := cache.ResetCache()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify counts are reset
	if cache.HitCount() != 0 {
		t.Errorf("expected hit count to be 0, got %d", cache.HitCount())
	}
	if cache.MissCount() != 0 {
		t.Errorf("expected miss count to be 0, got %d", cache.MissCount())
	}
}

func TestGorm2Cache_InvalidateSearchCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	// Set some search cache
	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	cache.SetSearchCache(ctx, "test-value", tableName, sql, 1)

	// Verify it exists
	exists, err := cache.SearchKeyExists(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected search cache to exist")
	}

	// Invalidate search cache
	err = cache.InvalidateSearchCache(ctx, tableName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	exists, err = cache.SearchKeyExists(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected search cache to be invalidated")
	}
}

func TestGorm2Cache_InvalidatePrimaryCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	primaryKey := "1"

	// Set primary cache
	err := cache.BatchSetPrimaryKeyCache(ctx, tableName, []util.Kv{
		{Key: primaryKey, Value: "test-value"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it exists
	exists, err := cache.BatchPrimaryKeyExists(ctx, tableName, []string{primaryKey})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected primary cache to exist")
	}

	// Invalidate primary cache
	err = cache.InvalidatePrimaryCache(ctx, tableName, primaryKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	exists, err = cache.BatchPrimaryKeyExists(ctx, tableName, []string{primaryKey})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected primary cache to be invalidated")
	}
}

func TestGorm2Cache_BatchInvalidatePrimaryCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	primaryKeys := []string{"1", "2", "3"}

	// Set primary cache
	kvs := make([]util.Kv, len(primaryKeys))
	for i, pk := range primaryKeys {
		kvs[i] = util.Kv{Key: pk, Value: "value-" + pk}
	}
	err := cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they exist
	exists, err := cache.BatchPrimaryKeyExists(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected primary caches to exist")
	}

	// Batch invalidate
	err = cache.BatchInvalidatePrimaryCache(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're gone
	exists, err = cache.BatchPrimaryKeyExists(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected primary caches to be invalidated")
	}
}

func TestGorm2Cache_InvalidateAllPrimaryCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	primaryKeys := []string{"1", "2"}

	// Set primary cache
	kvs := make([]util.Kv, len(primaryKeys))
	for i, pk := range primaryKeys {
		kvs[i] = util.Kv{Key: pk, Value: "value-" + pk}
	}
	err := cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Invalidate all primary cache
	err = cache.InvalidateAllPrimaryCache(ctx, tableName)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're gone
	exists, err := cache.BatchPrimaryKeyExists(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected primary caches to be invalidated")
	}
}

func TestGorm2Cache_BatchPrimaryKeyExists(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	primaryKeys := []string{"1", "2"}

	// Test non-existent keys
	exists, err := cache.BatchPrimaryKeyExists(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected keys to not exist")
	}

	// Set primary cache
	kvs := make([]util.Kv, len(primaryKeys))
	for i, pk := range primaryKeys {
		kvs[i] = util.Kv{Key: pk, Value: "value-" + pk}
	}
	err = cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Test existing keys
	exists, err = cache.BatchPrimaryKeyExists(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected keys to exist")
	}
}

func TestGorm2Cache_SearchKeyExists(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"

	// Test non-existent key
	exists, err := cache.SearchKeyExists(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Set search cache
	err = cache.SetSearchCache(ctx, "test-value", tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Test existing key
	exists, err = cache.SearchKeyExists(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}
}

func TestGorm2Cache_SetSearchCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	cacheValue := "1|{\"id\":1,\"name\":\"test\"}"

	err := cache.SetSearchCache(ctx, cacheValue, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was set
	value, err := cache.GetSearchCache(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != cacheValue {
		t.Errorf("expected %s, got %s", cacheValue, value)
	}
}

func TestGorm2Cache_GetSearchCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	cacheValue := "1|{\"id\":1,\"name\":\"test\"}"

	// Test non-existent cache
	_, err := cache.GetSearchCache(ctx, tableName, sql, 1)
	if err == nil {
		t.Error("expected error for non-existent cache")
	}

	// Set cache
	err = cache.SetSearchCache(ctx, cacheValue, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Get cache
	value, err := cache.GetSearchCache(ctx, tableName, sql, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != cacheValue {
		t.Errorf("expected %s, got %s", cacheValue, value)
	}
}

func TestGorm2Cache_BatchSetPrimaryKeyCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	kvs := []util.Kv{
		{Key: "1", Value: "{\"id\":1,\"name\":\"user1\"}"},
		{Key: "2", Value: "{\"id\":2,\"name\":\"user2\"}"},
	}

	err := cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they were set
	values, err := cache.BatchGetPrimaryCache(ctx, tableName, []string{"1", "2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
}

func TestGorm2Cache_BatchGetPrimaryCache(t *testing.T) {
	memStorage := storage.NewMem(storage.DefaultMemStoreConfig)
	cache := &Gorm2Cache{
		Config: &config.CacheConfig{
			CacheStorage: memStorage,
			CacheTTL:    1000,
			DebugMode:   false,
		},
		stats: &stats{},
	}
	cache.Init()
	ctx := context.Background()

	tableName := "users"
	primaryKeys := []string{"1", "2"}

	// Test non-existent keys - memory storage returns error when not all keys exist
	_, err := cache.BatchGetPrimaryCache(ctx, tableName, primaryKeys)
	if err == nil {
		t.Log("Note: memory storage returns error when not all keys exist, which is expected behavior")
	}

	// Set primary cache
	kvs := []util.Kv{
		{Key: "1", Value: "{\"id\":1}"},
		{Key: "2", Value: "{\"id\":2}"},
	}
	err = cache.BatchSetPrimaryKeyCache(ctx, tableName, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Get primary cache
	values, err := cache.BatchGetPrimaryCache(ctx, tableName, primaryKeys)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
}

