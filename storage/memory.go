package storage

import (
	"context"
	"fmt"
	"github.com/karlseguin/ccache/v3"
	"time"

	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/util"
)

type MemStoreConfig struct {
	MaxSize int64 // maximal items in primary cache
}

var DefaultMemStoreConfig = &MemStoreConfig{
	MaxSize: 1000,
}

func NewMem(config *MemStoreConfig) *Memory {
	return &Memory{config: config}
}

type Memory struct {
	config *MemStoreConfig

	cache *ccache.Cache[string]
	ttl   int64
}

func (m *Memory) Init(conf *config.CacheConfig, prefix string) error {
	c := ccache.New(ccache.Configure[string]().MaxSize(m.config.MaxSize))
	m.cache = c
	m.ttl = conf.CacheTTL
	return nil
}

func (m *Memory) CleanCache(ctx context.Context) error {
	m.cache.Clear()
	return nil
}

func (m *Memory) BatchKeyExist(ctx context.Context, keys []string) (bool, error) {
	for _, key := range keys {
		item := m.cache.Get(key)
		if item == nil || item.Expired() {
			return false, nil
		}
	}
	return true, nil
}

func (m *Memory) KeyExists(ctx context.Context, key string) (bool, error) {
	item := m.cache.Get(key)
	return item != nil && !item.Expired(), nil
}

func (m *Memory) GetValue(ctx context.Context, key string) (string, error) {
	item := m.cache.Get(key)
	if item == nil || item.Expired() {
		return "", ErrCacheNotFound
	}
	return item.Value(), nil
}

func (m *Memory) BatchGetValues(ctx context.Context, keys []string) ([]string, error) {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		item := m.cache.Get(key)
		if item != nil && !item.Expired() {
			values = append(values, item.Value())
		}
	}
	if len(values) != len(keys) {
		return nil, fmt.Errorf("cannot get items")
	}
	return values, nil
}

func (m *Memory) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	m.cache.DeletePrefix(keyPrefix)
	return nil
}

func (m *Memory) DeleteKey(ctx context.Context, key string) error {
	m.cache.Delete(key)
	return nil
}

func (m *Memory) BatchDeleteKeys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		m.cache.Delete(key)
	}
	return nil
}

func (m *Memory) BatchSetKeys(ctx context.Context, kvs []util.Kv) error {
	for _, kv := range kvs {
		if m.ttl > 0 {
			m.cache.Set(kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(m.ttl))*time.Millisecond)
		} else {
			m.cache.Set(kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(24))*time.Hour)
		}
	}
	return nil
}

func (m *Memory) SetKey(ctx context.Context, kv util.Kv) error {
	if m.ttl > 0 {
		m.cache.Set(kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(m.ttl))*time.Millisecond)
	} else {
		m.cache.Set(kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(24))*time.Hour)
	}
	return nil
}
