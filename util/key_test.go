package util

import (
	"reflect"
	"strings"
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
	
	tests := []struct {
		name           string
		primaryKeyVals []string
		expected       string
	}{
		{
			name:           "single primary key",
			primaryKeyVals: []string{"1"},
			expected:       "gormcache:test123:p:users:1",
		},
		{
			name:           "composite primary key",
			primaryKeyVals: []string{"1", "2"},
			expected:       "gormcache:test123:p:users:1:2",
		},
		{
			name:           "composite primary key with three fields",
			primaryKeyVals: []string{"1", "2", "3"},
			expected:       "gormcache:test123:p:users:1:2:3",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenPrimaryCacheKey(instanceId, tableName, tt.primaryKeyVals...)
			if key != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, key)
			}
		})
	}
}

func TestGenPrimaryCacheKeyFromMap(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	
	tests := []struct {
		name         string
		primaryKeyMap map[string]string
		expected     string
	}{
		{
			name: "single primary key",
			primaryKeyMap: map[string]string{
				"id": "1",
			},
			expected: "gormcache:test123:p:users:1",
		},
		{
			name: "composite primary key",
			primaryKeyMap: map[string]string{
				"user_id": "1",
				"role_id": "2",
			},
			expected: "gormcache:test123:p:users:1:2", // sorted by field name: role_id, user_id
		},
		{
			name: "composite primary key with three fields",
			primaryKeyMap: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			expected: "gormcache:test123:p:users:1:2:3", // sorted: a, b, c
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenPrimaryCacheKeyFromMap(instanceId, tableName, tt.primaryKeyMap)
			// 验证key格式正确，包含所有值（顺序可能因map遍历顺序而不同，但排序后应该一致）
			if len(key) == 0 {
				t.Error("expected non-empty key")
			}
			// 验证前缀正确
			expectedPrefix := "gormcache:test123:p:users:"
			if !strings.Contains(key, expectedPrefix) {
				t.Errorf("key should contain prefix %s, got %s", expectedPrefix, key)
			}
			// 对于单个key，验证完整匹配
			if len(tt.primaryKeyMap) == 1 {
				if key != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, key)
				}
			} else {
				// 对于多个key，验证包含所有值
				for _, val := range tt.primaryKeyMap {
					if !strings.Contains(key, val) {
						t.Errorf("key should contain value %s, got %s", val, key)
					}
				}
			}
		})
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

func TestGenUniqueCacheKey(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	uniqueIndexName := "idx_email"
	
	tests := []struct {
		name           string
		uniqueKeyVals  []string
		expected       string
	}{
		{
			name:          "single unique key",
			uniqueKeyVals: []string{"user@example.com"},
			expected:      "gormcache:test123:u:users:idx_email:user@example.com",
		},
		{
			name:          "composite unique key",
			uniqueKeyVals: []string{"user@example.com", "123"},
			expected:      "gormcache:test123:u:users:idx_email:user@example.com:123",
		},
		{
			name:          "composite unique key with three fields",
			uniqueKeyVals: []string{"1", "2", "3"},
			expected:      "gormcache:test123:u:users:idx_email:1:2:3",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenUniqueCacheKey(instanceId, tableName, uniqueIndexName, tt.uniqueKeyVals...)
			if key != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, key)
			}
		})
	}
}

func TestGenUniqueCacheKeyFromMap(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	uniqueIndexName := "idx_email"
	
	tests := []struct {
		name         string
		uniqueKeyMap map[string]string
		expected     string
	}{
		{
			name: "single unique key",
			uniqueKeyMap: map[string]string{
				"email": "user@example.com",
			},
			expected: "gormcache:test123:u:users:idx_email:user@example.com",
		},
		{
			name: "composite unique key",
			uniqueKeyMap: map[string]string{
				"email": "user@example.com",
				"code":  "123",
			},
			expected: "gormcache:test123:u:users:idx_email:123:user@example.com", // sorted: code, email
		},
		{
			name: "composite unique key with three fields",
			uniqueKeyMap: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			expected: "gormcache:test123:u:users:idx_email:1:2:3", // sorted: a, b, c
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := GenUniqueCacheKeyFromMap(instanceId, tableName, uniqueIndexName, tt.uniqueKeyMap)
			// 验证key格式正确
			if len(key) == 0 {
				t.Error("expected non-empty key")
			}
			// 验证前缀正确
			expectedPrefix := "gormcache:test123:u:users:idx_email:"
			if !strings.Contains(key, expectedPrefix) {
				t.Errorf("key should contain prefix %s, got %s", expectedPrefix, key)
			}
			// 对于单个key，验证完整匹配
			if len(tt.uniqueKeyMap) == 1 {
				if key != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, key)
				}
			} else {
				// 对于多个key，验证包含所有值
				for _, val := range tt.uniqueKeyMap {
					if !strings.Contains(key, val) {
						t.Errorf("key should contain value %s, got %s", val, key)
					}
				}
			}
		})
	}
}

func TestGenUniqueCachePrefix(t *testing.T) {
	instanceId := "test123"
	tableName := "users"
	uniqueIndexName := "idx_email"
	
	prefix := GenUniqueCachePrefix(instanceId, tableName, uniqueIndexName)
	expected := "gormcache:test123:u:users:idx_email"
	
	if prefix != expected {
		t.Errorf("expected %s, got %s", expected, prefix)
	}
}

