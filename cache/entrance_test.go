package cache

import (
	"testing"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/storage"
)

func TestNewGorm2Cache(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.CacheConfig
		expectError bool
		checkFunc   func(*Gorm2Cache) error
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name: "valid config with storage",
			config: &config.CacheConfig{
				CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
				CacheTTL:     1000,
				DebugMode:    false,
			},
			expectError: false,
			checkFunc: func(c *Gorm2Cache) error {
				if c == nil {
					return &testError{msg: "expected non-nil cache"}
				}
				if c.Config == nil {
					return &testError{msg: "expected config to be set"}
				}
				if c.InstanceId == "" {
					return &testError{msg: "expected InstanceId to be set"}
				}
				return nil
			},
		},
		{
			name: "valid config without storage",
			config: &config.CacheConfig{
				CacheTTL:  1000,
				DebugMode: false,
			},
			expectError: false,
			checkFunc: func(c *Gorm2Cache) error {
				if c == nil {
					return &testError{msg: "expected non-nil cache"}
				}
				if c.InstanceId == "" {
					return &testError{msg: "expected InstanceId to be set"}
				}
				return nil
			},
		},
		{
			name: "valid config with debug logger",
			config: &config.CacheConfig{
				CacheStorage: storage.NewMem(storage.DefaultMemStoreConfig),
				CacheTTL:     1000,
				DebugMode:    true,
			},
			expectError: false,
			checkFunc: func(c *Gorm2Cache) error {
				if c.Logger == nil {
					return &testError{msg: "expected Logger to be set"}
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewGorm2Cache(tt.config)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
				return
			}
			if tt.expectError {
				return
			}
			if cache == nil {
				t.Fatal("expected non-nil cache")
			}
			if tt.checkFunc != nil {
				if gormCache, ok := cache.(*Gorm2Cache); ok {
					if err := tt.checkFunc(gormCache); err != nil {
						t.Error(err)
					}
				} else {
					t.Error("expected cache to be *Gorm2Cache")
				}
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

