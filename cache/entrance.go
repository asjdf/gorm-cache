package cache

import (
	"fmt"

	"github.com/asjdf/gorm-cache/config"
)

func NewGorm2Cache(cacheConfig *config.CacheConfig) (Cache, error) {
	if cacheConfig == nil {
		return nil, fmt.Errorf("you pass a nil config")
	}
	cache := &Gorm2Cache{
		Config: cacheConfig,
		stats:  &stats{},
	}
	err := cache.Init()
	if err != nil {
		return nil, err
	}
	return cache, nil
}
