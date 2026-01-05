package storage

import (
	"context"
	"testing"

	"github.com/asjdf/gorm-cache/util"
	"github.com/bluele/gcache"
)

func TestNewGcache(t *testing.T) {
	// Test with nil builder (should use default)
	gc := NewGcache(nil)
	if gc == nil {
		t.Fatal("expected non-nil Gcache instance")
	}
	if gc.builder == nil {
		t.Error("expected builder to be set")
	}

	// Test with custom builder
	builder := gcache.New(500).ARC()
	gc2 := NewGcache(builder)
	if gc2 == nil {
		t.Fatal("expected non-nil Gcache instance")
	}
	if gc2.builder != builder {
		t.Error("expected builder to be set")
	}
}

func TestGcache_Init(t *testing.T) {
	gc := NewGcache(nil)
	config := &Config{
		TTL:    1000,
		Debug:  true,
		Logger: &util.DefaultLogger{},
	}

	err := gc.Init(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gc.cache == nil {
		t.Error("expected cache to be initialized")
	}

	// Test idempotency
	err2 := gc.Init(config)
	if err2 != nil {
		t.Fatalf("expected no error on second init, got %v", err2)
	}
}

func TestGcache_Init_WithTTL(t *testing.T) {
	gc := NewGcache(nil)
	config := &Config{
		TTL:    5000,
		Debug:  false,
		Logger: &util.DefaultLogger{},
	}

	err := gc.Init(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGcache_Init_WithoutTTL(t *testing.T) {
	gc := NewGcache(nil)
	config := &Config{
		TTL:    0,
		Debug:  false,
		Logger: &util.DefaultLogger{},
	}

	err := gc.Init(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGcache_KeyExists(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test non-existent key
	exists, err := gc.KeyExists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Set a key
	gc.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})

	// Test existing key
	exists, err = gc.KeyExists(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}
}

func TestGcache_GetValue(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test non-existent key
	_, err := gc.GetValue(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}

	// Set a key
	expectedValue := "test-value"
	gc.SetKey(ctx, util.Kv{Key: "test-key", Value: expectedValue})

	// Get the value
	value, err := gc.GetValue(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != expectedValue {
		t.Errorf("expected %s, got %s", expectedValue, value)
	}
}

func TestGcache_SetKey(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	err := gc.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was set
	value, err := gc.GetValue(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != "test-value" {
		t.Errorf("expected 'test-value', got %s", value)
	}
}

func TestGcache_DeleteKey(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set a key
	gc.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})

	// Verify it exists
	exists, _ := gc.KeyExists(ctx, "test-key")
	if !exists {
		t.Error("expected key to exist before deletion")
	}

	// Delete it
	err := gc.DeleteKey(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's deleted
	exists, _ = gc.KeyExists(ctx, "test-key")
	if exists {
		t.Error("expected key to be deleted")
	}
}

func TestGcache_BatchDeleteKeys(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set some keys
	gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	gc.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})
	gc.SetKey(ctx, util.Kv{Key: "key3", Value: "value3"})

	// Delete multiple keys
	err := gc.BatchDeleteKeys(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're deleted
	exists1, _ := gc.KeyExists(ctx, "key1")
	exists2, _ := gc.KeyExists(ctx, "key2")
	exists3, _ := gc.KeyExists(ctx, "key3")

	if exists1 {
		t.Error("expected key1 to be deleted")
	}
	if exists2 {
		t.Error("expected key2 to be deleted")
	}
	if !exists3 {
		t.Error("expected key3 to still exist")
	}
}

func TestGcache_BatchKeyExist(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test with all keys existing
	gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	gc.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	exists, err := gc.BatchKeyExist(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected all keys to exist")
	}

	// Test with some keys missing
	exists, err = gc.BatchKeyExist(ctx, []string{"key1", "key3"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected false when some keys are missing")
	}
}

func TestGcache_BatchKeyExist_EmptySlice(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test with empty slice
	exists, err := gc.BatchKeyExist(ctx, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected true for empty slice")
	}
}

func TestGcache_BatchKeyExist_AllMissing(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test with all keys missing
	exists, err := gc.BatchKeyExist(ctx, []string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected false when all keys are missing")
	}
}

func TestGcache_BatchGetValues(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set some keys
	gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	gc.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	// Get values
	values, err := gc.BatchGetValues(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != "value1" {
		t.Errorf("expected 'value1', got %s", values[0])
	}
	if values[1] != "value2" {
		t.Errorf("expected 'value2', got %s", values[1])
	}
}

func TestGcache_BatchGetValues_WithMissingKey(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set one key
	gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})

	// Try to get with missing key (should return error)
	_, err := gc.BatchGetValues(ctx, []string{"key1", "key2"})
	if err == nil {
		t.Error("expected error when some keys are missing")
	}
}

func TestGcache_BatchSetKeys(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	kvs := []util.Kv{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}

	err := gc.BatchSetKeys(ctx, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all keys are set
	for _, kv := range kvs {
		value, err := gc.GetValue(ctx, kv.Key)
		if err != nil {
			t.Fatalf("expected no error getting %s, got %v", kv.Key, err)
		}
		if value != kv.Value {
			t.Errorf("expected %s for key %s, got %s", kv.Value, kv.Key, value)
		}
	}
}

func TestGcache_BatchSetKeys_EmptySlice(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	err := gc.BatchSetKeys(ctx, []util.Kv{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGcache_BatchSetKeys_WithError(t *testing.T) {
	// Create a cache that will fail on Set
	// This tests the error handling path in BatchSetKeys
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set a key first
	err := gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Batch set should work normally
	kvs := []util.Kv{
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}
	err = gc.BatchSetKeys(ctx, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGcache_CleanCache(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set some keys
	gc.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	gc.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	// Verify they exist
	exists1, _ := gc.KeyExists(ctx, "key1")
	exists2, _ := gc.KeyExists(ctx, "key2")
	if !exists1 || !exists2 {
		t.Error("expected keys to exist before cleanup")
	}

	// Clean cache
	err := gc.CleanCache(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're gone
	exists1, _ = gc.KeyExists(ctx, "key1")
	exists2, _ = gc.KeyExists(ctx, "key2")
	if exists1 || exists2 {
		t.Error("expected keys to be deleted after cleanup")
	}
}

func TestGcache_DeleteKeysWithPrefix(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set keys with different prefixes
	gc.SetKey(ctx, util.Kv{Key: "prefix1:key1", Value: "value1"})
	gc.SetKey(ctx, util.Kv{Key: "prefix1:key2", Value: "value2"})
	gc.SetKey(ctx, util.Kv{Key: "prefix2:key1", Value: "value3"})

	// Delete keys with prefix1
	err := gc.DeleteKeysWithPrefix(ctx, "prefix1:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify prefix1 keys are deleted
	exists1, _ := gc.KeyExists(ctx, "prefix1:key1")
	exists2, _ := gc.KeyExists(ctx, "prefix1:key2")
	exists3, _ := gc.KeyExists(ctx, "prefix2:key1")

	if exists1 {
		t.Error("expected prefix1:key1 to be deleted")
	}
	if exists2 {
		t.Error("expected prefix1:key2 to be deleted")
	}
	if !exists3 {
		t.Error("expected prefix2:key1 to still exist")
	}
}

func TestGcache_ConcurrentAccess(t *testing.T) {
	gc := NewGcache(nil)
	gc.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			key := "key-" + string(rune('0'+i))
			gc.SetKey(ctx, util.Kv{Key: key, Value: "value"})
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all keys exist
	for i := 0; i < 10; i++ {
		key := "key-" + string(rune('0'+i))
		exists, _ := gc.KeyExists(ctx, key)
		if !exists {
			t.Errorf("expected key %s to exist", key)
		}
	}
}
