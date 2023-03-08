package storage

import (
	"context"
	"errors"
	"github.com/asjdf/gorm-cache/util"
)

var (
	ErrCacheNotFound = errors.New("cache not found")
)

type Config struct {
	TTL    int64
	Debug  bool
	Logger util.LoggerInterface
}

type DataStorage interface {
	Init(config *Config) error
	CleanCache(ctx context.Context) error

	// read
	BatchKeyExist(ctx context.Context, keys []string) (bool, error)
	KeyExists(ctx context.Context, key string) (bool, error)
	GetValue(ctx context.Context, key string) (string, error)
	BatchGetValues(ctx context.Context, keys []string) ([]string, error)

	// write
	DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error
	DeleteKey(ctx context.Context, key string) error
	BatchDeleteKeys(ctx context.Context, keys []string) error
	BatchSetKeys(ctx context.Context, kvs []util.Kv) error
	SetKey(ctx context.Context, kv util.Kv) error
}
