package util

import (
	"reflect"
	"testing"
)

func TestGenInstanceId(t *testing.T) {
	id1 := GenInstanceId()
	id2 := GenInstanceId()
	
	if len(id1) != 5 {
		t.Errorf("expected instance id length to be 5, got %d", len(id1))
	}
	if len(id2) != 5 {
		t.Errorf("expected instance id length to be 5, got %d", len(id2))
	}
	// IDs should be different (very high probability)
	if id1 == id2 {
		t.Error("expected different instance ids")
	}
}

func TestGenPrimaryCacheKey(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	primaryKey := "1"
	
	key := GenPrimaryCacheKey(instanceId, tableName, primaryKey)
	expected := "gormcache:test123:p:users:1"
	
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestGenPrimaryCachePrefix(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	
	prefix := GenPrimaryCachePrefix(instanceId, tableName)
	expected := "gormcache:test123:p:users"
	
	if prefix != expected {
		t.Errorf("expected %s, got %s", expected, prefix)
	}
}

func TestGenSearchCacheKey(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	
	tests := []struct {
		name     string
		vars     []interface{}
		expected string
	}{
		{
			name:     "no vars",
			vars:     []interface{}{},
			expected: "gormcache:test123:s:users:SELECT * FROM users WHERE id = ?",
		},
		{
			name:     "with int var",
			vars:     []interface{}{1},
			expected: "gormcache:test123:s:users:SELECT * FROM users WHERE id = ?:1",
		},
		{
			name:     "with string var",
			vars:     []interface{}{"test"},
			expected: "gormcache:test123:s:users:SELECT * FROM users WHERE id = ?:test",
		},
		{
			name:     "with pointer var",
			vars:     []interface{}{intPtr(42)},
			expected: "gormcache:test123:s:users:SELECT * FROM users WHERE id = ?:42",
		},
		{
			name:     "with multiple vars",
			vars:     []interface{}{1, "test", intPtr(100)},
			expected: "gormcache:test123:s:users:SELECT * FROM users WHERE id = ?:1:test:100",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenSearchCacheKey(instanceId, tableName, sql, tt.vars...)
			if key != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, key)
			}
		})
	}
}

func TestGenSearchCachePrefix(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	
	prefix := GenSearchCachePrefix(instanceId, tableName)
	expected := "gormcache:test123:s:users"
	
	if prefix != expected {
		t.Errorf("expected %s, got %s", expected, prefix)
	}
}

func TestGenSingleFlightKey(t *testing.T) {
	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	
	tests := []struct {
		name     string
		vars     []interface{}
		expected string
	}{
		{
			name:     "no vars",
			vars:     []interface{}{},
			expected: "users:SELECT * FROM users WHERE id = ?",
		},
		{
			name:     "with int var",
			vars:     []interface{}{1},
			expected: "users:SELECT * FROM users WHERE id = ?:1",
		},
		{
			name:     "with string var",
			vars:     []interface{}{"test"},
			expected: "users:SELECT * FROM users WHERE id = ?:test",
		},
		{
			name:     "with pointer var",
			vars:     []interface{}{intPtr(42)},
			expected: "users:SELECT * FROM users WHERE id = ?:42",
		},
		{
			name:     "with multiple vars",
			vars:     []interface{}{1, "test", intPtr(100)},
			expected: "users:SELECT * FROM users WHERE id = ?:1:test:100",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenSingleFlightKey(tableName, sql, tt.vars...)
			if key != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, key)
			}
		})
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

func TestGenSearchCacheKey_WithComplexTypes(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	sql := "SELECT * FROM users"
	
	// Test with different types
	var intVal int = 42
	var strVal string = "test"
	var boolVal bool = true
	
	vars := []interface{}{&intVal, strVal, boolVal}
	key := GenSearchCacheKey(instanceId, tableName, sql, vars...)
	
	// Just verify it doesn't panic and produces a key
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestGenSingleFlightKey_WithComplexTypes(t *testing.T) {
	tableName := "users"
	sql := "SELECT * FROM users"
	
	// Test with different types
	var intVal int = 42
	var strVal string = "test"
	var boolVal bool = true
	
	vars := []interface{}{&intVal, strVal, boolVal}
	key := GenSingleFlightKey(tableName, sql, vars...)
	
	// Just verify it doesn't panic and produces a key
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestGenSearchCacheKey_WithNilPointer(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	
	var nilPtr *int = nil
	vars := []interface{}{nilPtr}
	
	// Should not panic
	key := GenSearchCacheKey(instanceId, tableName, sql, vars...)
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestGenSingleFlightKey_WithNilPointer(t *testing.T) {
	tableName := "users"
	sql := "SELECT * FROM users WHERE id = ?"
	
	var nilPtr *int = nil
	vars := []interface{}{nilPtr}
	
	// Should not panic
	key := GenSingleFlightKey(tableName, sql, vars...)
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

func TestGenSearchCacheKey_ReflectValue(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	sql := "SELECT * FROM users"
	
	// Test with reflect.Value
	rv := reflect.ValueOf(42)
	vars := []interface{}{rv}
	
	key := GenSearchCacheKey(instanceId, tableName, sql, vars...)
	if len(key) == 0 {
		t.Error("expected non-empty key")
	}
}

