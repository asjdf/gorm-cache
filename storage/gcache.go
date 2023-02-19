package storage

import (
	"context"
	"github.com/asjdf/gorm-cache/config"
	"github.com/asjdf/gorm-cache/util"
	"github.com/bluele/gcache"
	"strings"
	"time"
)

var _ DataStorage = &Gcache{}

func NewGcache(builder *gcache.CacheBuilder) *Gcache {
	if builder == nil {
		builder = gcache.New(1000).ARC()
	}
	return &Gcache{builder: builder}
}

type Gcache struct {
	builder *gcache.CacheBuilder
	cache   gcache.Cache
}

func (g *Gcache) Init(config *config.CacheConfig, prefix string) error {
	if config.CacheTTL != 0 {
		g.builder.Expiration(time.Duration(config.CacheTTL) * time.Microsecond)
	}
	g.cache = g.builder.Build()
	return nil
}

func (g *Gcache) BatchKeyExist(ctx context.Context, keys []string) (bool, error) {
	for _, key := range keys {
		if !g.cache.Has(key) {
			return false, nil
		}
	}
	return true, nil
}

func (g *Gcache) KeyExists(ctx context.Context, key string) (bool, error) {
	return g.cache.Has(key), nil
}

func (g *Gcache) GetValue(ctx context.Context, key string) (string, error) {
	v, err := g.cache.Get(key)
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (g *Gcache) BatchGetValues(ctx context.Context, keys []string) ([]string, error) {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		v, err := g.cache.Get(key)
		if err != nil {
			return nil, err
		}
		values = append(values, v.(string))
	}
	return values, nil
}

func (g *Gcache) CleanCache(ctx context.Context) error {
	g.cache.Purge()
	return nil
}

func (g *Gcache) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	all := g.cache.GetALL(false)
	for k, _ := range all {
		if key, ok := k.(string); ok && strings.HasPrefix(key, keyPrefix) {
			g.cache.Remove(key)
		}
	}
	return nil
}

func (g *Gcache) DeleteKey(ctx context.Context, key string) error {
	g.cache.Remove(key)
	return nil
}

func (g *Gcache) BatchDeleteKeys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		g.cache.Remove(key)
	}
	return nil
}

func (g *Gcache) BatchSetKeys(ctx context.Context, kvs []util.Kv) error {
	for _, kv := range kvs {
		if err := g.SetKey(ctx, kv); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gcache) SetKey(ctx context.Context, kv util.Kv) error {
	return g.cache.Set(kv.Key, kv.Value)
}
