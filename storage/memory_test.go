package storage

import (
	"context"
	"testing"
	"time"

	"github.com/asjdf/gorm-cache/util"
)

func TestNewMem(t *testing.T) {
	// Test with no config
	mem := NewMem()
	if mem == nil {
		t.Fatal("expected non-nil Memory instance")
	}
	if mem.config == nil {
		t.Fatal("expected non-nil config")
	}
	if mem.config.MaxSize != 1000 {
		t.Errorf("expected default MaxSize to be 1000, got %d", mem.config.MaxSize)
	}

	// Test with custom config
	customConfig := &MemStoreConfig{MaxSize: 500}
	mem2 := NewMem(customConfig)
	if mem2.config.MaxSize != 500 {
		t.Errorf("expected MaxSize to be 500, got %d", mem2.config.MaxSize)
	}
}

func TestMemory_Init(t *testing.T) {
	mem := NewMem()
	logger := &util.DefaultLogger{}
	config := &Config{
		TTL:    1000,
		Debug:  true,
		Logger: logger,
	}

	err := mem.Init(config)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mem.cache == nil {
		t.Fatal("expected cache to be initialized")
	}
	if mem.ttl != 1000 {
		t.Errorf("expected ttl to be 1000, got %d", mem.ttl)
	}

	// Test that Init is idempotent (once.Do)
	err2 := mem.Init(config)
	if err2 != nil {
		t.Fatalf("expected no error on second init, got %v", err2)
	}
}

func TestMemory_CleanCache(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set some values
	mem.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	mem.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	// Verify they exist
	exists, _ := mem.KeyExists(ctx, "key1")
	if !exists {
		t.Error("expected key1 to exist")
	}

	// Clean cache
	err := mem.CleanCache(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're gone
	exists, _ = mem.KeyExists(ctx, "key1")
	if exists {
		t.Error("expected key1 to be deleted")
	}
}

func TestMemory_KeyExists(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test non-existent key
	exists, err := mem.KeyExists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected key to not exist")
	}

	// Set a key
	mem.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})

	// Test existing key
	exists, err = mem.KeyExists(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}
}

func TestMemory_GetValue(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test non-existent key
	_, err := mem.GetValue(ctx, "nonexistent")
	if err != ErrCacheNotFound {
		t.Errorf("expected ErrCacheNotFound, got %v", err)
	}

	// Set a key
	expectedValue := "test-value"
	mem.SetKey(ctx, util.Kv{Key: "test-key", Value: expectedValue})

	// Get the value
	value, err := mem.GetValue(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != expectedValue {
		t.Errorf("expected %s, got %s", expectedValue, value)
	}
}

func TestMemory_BatchKeyExist(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test empty keys
	exists, err := mem.BatchKeyExist(ctx, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected empty keys to return true")
	}

	// Test all keys exist
	mem.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	mem.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	exists, err = mem.BatchKeyExist(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected all keys to exist")
	}

	// Test some keys missing
	exists, err = mem.BatchKeyExist(ctx, []string{"key1", "key3"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected false when some keys are missing")
	}
}

func TestMemory_BatchKeyExist_AllMissing(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test all keys missing
	exists, err := mem.BatchKeyExist(ctx, []string{"nonexistent1", "nonexistent2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected false when all keys are missing")
	}
}

func TestMemory_BatchKeyExist_WithExpiredKey(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 10, Logger: &util.DefaultLogger{}}) // Very short TTL
	ctx := context.Background()

	// Set a key
	mem.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})

	// Verify it exists immediately
	exists, err := mem.BatchKeyExist(ctx, []string{"key1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Error("expected key to exist immediately")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Verify it's expired
	exists, err = mem.BatchKeyExist(ctx, []string{"key1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Error("expected key to be expired")
	}
}

func TestMemory_BatchGetValues(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Test empty keys
	values, err := mem.BatchGetValues(ctx, []string{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty values, got %d", len(values))
	}

	// Set some keys
	mem.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	mem.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})

	// Test getting all values
	values, err = mem.BatchGetValues(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(values))
	}
	if values[0] != "value1" {
		t.Errorf("expected value1, got %s", values[0])
	}
	if values[1] != "value2" {
		t.Errorf("expected value2, got %s", values[1])
	}

	// Test getting with missing key (should return error)
	_, err = mem.BatchGetValues(ctx, []string{"key1", "key3"})
	if err == nil {
		t.Error("expected error when some keys are missing")
	}
}

func TestMemory_DeleteKey(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set a key
	mem.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})

	// Verify it exists
	exists, _ := mem.KeyExists(ctx, "test-key")
	if !exists {
		t.Error("expected key to exist before deletion")
	}

	// Delete it
	err := mem.DeleteKey(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it's gone
	exists, _ = mem.KeyExists(ctx, "test-key")
	if exists {
		t.Error("expected key to be deleted")
	}

	// Delete non-existent key (should not error)
	err = mem.DeleteKey(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("expected no error deleting non-existent key, got %v", err)
	}
}

func TestMemory_BatchDeleteKeys(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set some keys
	mem.SetKey(ctx, util.Kv{Key: "key1", Value: "value1"})
	mem.SetKey(ctx, util.Kv{Key: "key2", Value: "value2"})
	mem.SetKey(ctx, util.Kv{Key: "key3", Value: "value3"})

	// Delete multiple keys
	err := mem.BatchDeleteKeys(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify they're deleted
	exists1, _ := mem.KeyExists(ctx, "key1")
	exists2, _ := mem.KeyExists(ctx, "key2")
	exists3, _ := mem.KeyExists(ctx, "key3")

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

func TestMemory_DeleteKeysWithPrefix(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set keys with different prefixes
	mem.SetKey(ctx, util.Kv{Key: "prefix1:key1", Value: "value1"})
	mem.SetKey(ctx, util.Kv{Key: "prefix1:key2", Value: "value2"})
	mem.SetKey(ctx, util.Kv{Key: "prefix2:key1", Value: "value3"})

	// Delete keys with prefix1
	err := mem.DeleteKeysWithPrefix(ctx, "prefix1:")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify prefix1 keys are deleted
	exists1, _ := mem.KeyExists(ctx, "prefix1:key1")
	exists2, _ := mem.KeyExists(ctx, "prefix1:key2")
	exists3, _ := mem.KeyExists(ctx, "prefix2:key1")

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

func TestMemory_SetKey(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	// Set a key
	err := mem.SetKey(ctx, util.Kv{Key: "test-key", Value: "test-value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it exists
	value, err := mem.GetValue(ctx, "test-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if value != "test-value" {
		t.Errorf("expected test-value, got %s", value)
	}
}

func TestMemory_BatchSetKeys(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	kvs := []util.Kv{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}

	err := mem.BatchSetKeys(ctx, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all keys are set
	for _, kv := range kvs {
		value, err := mem.GetValue(ctx, kv.Key)
		if err != nil {
			t.Fatalf("expected no error getting %s, got %v", kv.Key, err)
		}
		if value != kv.Value {
			t.Errorf("expected %s for key %s, got %s", kv.Value, kv.Key, value)
		}
	}
}

func TestMemory_BatchSetKeys_WithZeroTTL(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 0, Logger: &util.DefaultLogger{}}) // TTL = 0
	ctx := context.Background()

	kvs := []util.Kv{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}

	err := mem.BatchSetKeys(ctx, kvs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify all keys are set (should use 24h TTL when TTL is 0)
	for _, kv := range kvs {
		value, err := mem.GetValue(ctx, kv.Key)
		if err != nil {
			t.Fatalf("expected no error getting %s, got %v", kv.Key, err)
		}
		if value != kv.Value {
			t.Errorf("expected %s for key %s, got %s", kv.Value, kv.Key, value)
		}
	}
}

func TestMemory_BatchSetKeys_EmptySlice(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 1000, Logger: &util.DefaultLogger{}})
	ctx := context.Background()

	err := mem.BatchSetKeys(ctx, []util.Kv{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMemory_SetKey_WithTTL(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 100, Logger: &util.DefaultLogger{}}) // 100ms TTL
	ctx := context.Background()

	// Set a key with TTL
	err := mem.SetKey(ctx, util.Kv{Key: "ttl-key", Value: "ttl-value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it exists immediately
	exists, _ := mem.KeyExists(ctx, "ttl-key")
	if !exists {
		t.Error("expected key to exist immediately")
	}

	// Wait for TTL to expire (with some buffer)
	time.Sleep(200 * time.Millisecond)

	// Verify it's expired
	exists, _ = mem.KeyExists(ctx, "ttl-key")
	if exists {
		t.Error("expected key to be expired")
	}
}

func TestMemory_SetKey_NoTTL(t *testing.T) {
	mem := NewMem()
	mem.Init(&Config{TTL: 0, Logger: &util.DefaultLogger{}}) // No TTL
	ctx := context.Background()

	// Set a key without TTL
	err := mem.SetKey(ctx, util.Kv{Key: "no-ttl-key", Value: "no-ttl-value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Verify it still exists (should have 24h TTL)
	exists, _ := mem.KeyExists(ctx, "no-ttl-key")
	if !exists {
		t.Error("expected key to still exist")
	}
}
