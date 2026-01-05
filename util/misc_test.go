package util

import (
	"testing"
)

func TestShouldCache(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		tables    []string
		expected  bool
	}{
		{
			name:      "empty tables list should cache all",
			tableName: "users",
			tables:    []string{},
			expected:  true,
		},
		{
			name:      "nil tables list should cache all",
			tableName: "users",
			tables:    nil,
			expected:  true,
		},
		{
			name:      "table in list should cache",
			tableName: "users",
			tables:    []string{"users", "posts"},
			expected:  true,
		},
		{
			name:      "table not in list should not cache",
			tableName: "comments",
			tables:    []string{"users", "posts"},
			expected:  false,
		},
		{
			name:      "case sensitive match",
			tableName: "Users",
			tables:    []string{"users"},
			expected:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldCache(tt.tableName, tt.tables)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestContainString(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		slice    []string
		expected bool
	}{
		{
			name:     "string in slice",
			target:   "test",
			slice:    []string{"test", "other"},
			expected: true,
		},
		{
			name:     "string not in slice",
			target:   "test",
			slice:    []string{"other", "another"},
			expected: false,
		},
		{
			name:     "empty slice",
			target:   "test",
			slice:    []string{},
			expected: false,
		},
		{
			name:     "nil slice",
			target:   "test",
			slice:    nil,
			expected: false,
		},
		{
			name:     "case sensitive",
			target:   "Test",
			slice:    []string{"test"},
			expected: false,
		},
		{
			name:     "empty string in slice",
			target:   "",
			slice:    []string{"", "other"},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainString(tt.target, tt.slice)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRandFloatingInt64(t *testing.T) {
	tests := []struct {
		name     string
		v        int64
		expected bool // whether result is in expected range
	}{
		{
			name:     "zero value",
			v:        0,
			expected: true, // should return 0
		},
		{
			name:     "positive value",
			v:        1000,
			expected: true, // should be between 900 and 1100 (0.9 * 1000 to 1.1 * 1000)
		},
		{
			name:     "large value",
			v:        1000000,
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandFloatingInt64(tt.v)
			
			if tt.v == 0 {
				if result != 0 {
					t.Errorf("expected 0 for zero input, got %d", result)
				}
				return
			}
			
			// Result should be between 0.9 * v and 1.1 * v
			min := int64(float64(tt.v) * 0.9)
			max := int64(float64(tt.v) * 1.1)
			
			if result < min || result > max {
				t.Errorf("expected result between %d and %d, got %d", min, max, result)
			}
		})
	}
	
	// Test multiple calls produce different results (with high probability)
	v := int64(1000)
	results := make(map[int64]bool)
	for i := 0; i < 100; i++ {
		result := RandFloatingInt64(v)
		results[result] = true
	}
	
	// With 100 calls, we should get some variation
	if len(results) < 2 {
		t.Log("warning: RandFloatingInt64 might not be random enough, but this is not necessarily an error")
	}
}

