package storage

import (
	"context"
	"sync"
	"time"

	"github.com/asjdf/gorm-cache/util"
	"github.com/redis/go-redis/v9"
)

var _ DataStorage = &Redis{}

type RedisStoreConfig struct {
	KeyPrefix string // key prefix will be random if not set

	Client  *redis.Client // if Client is not nil, Options will be ignored
	Options *redis.Options
}

func NewRedis(config ...*RedisStoreConfig) *Redis {
	if len(config) == 0 {
		panic("redis config is required")
	}
	if config[0].KeyPrefix == "" {
		config[0].KeyPrefix = util.GormCachePrefix + ":" + util.GenInstanceId()
	}
	r := &Redis{
		keyPrefix: config[0].KeyPrefix,
	}
	if config[0].Client != nil {
		r.client = config[0].Client
		return r
	}
	r.client = redis.NewClient(config[0].Options)
	return r
}

type Redis struct {
	client    *redis.Client
	ttl       int64
	logger    util.LoggerInterface
	keyPrefix string

	batchExistSha string
	cleanCacheSha string

	once sync.Once
}

func (r *Redis) Init(conf *Config) error {
	var err error
	r.once.Do(func() {
		r.ttl = conf.TTL
		r.logger = conf.Logger
		r.logger.SetIsDebug(conf.Debug)
		err = r.initScripts()
	})
	return err
}

func (r *Redis) initScripts() error {
	batchKeyExistScript := `
		for idx, val in pairs(KEYS) do
			local exists = redis.call('EXISTS', val)
			if exists == 0 then
				return 0
			end
		end
		return 1`

	cleanCacheScript := `
		local keys = redis.call('keys', ARGV[1])
		for i=1,#keys,5000 do 
			redis.call('del', 'defaultKey', unpack(keys, i, math.min(i+4999, #keys)))
		end
		return 1`

	result := r.client.ScriptLoad(context.Background(), batchKeyExistScript)
	if result.Err() != nil {
		r.logger.CtxError(context.Background(), "[initScripts] init script 1 error: %v", result.Err())
		return result.Err()
	}
	r.batchExistSha = result.Val()
	r.logger.CtxInfo(context.Background(), "[initScripts] init batch exist script sha1: %s", r.batchExistSha)

	result = r.client.ScriptLoad(context.Background(), cleanCacheScript)
	if result.Err() != nil {
		r.logger.CtxError(context.Background(), "[initScripts] init script 2 error: %v", result.Err())
		return result.Err()
	}
	r.cleanCacheSha = result.Val()
	r.logger.CtxInfo(context.Background(), "[initScripts] init clean cache script sha1: %s", r.cleanCacheSha)
	return nil
}

func (r *Redis) CleanCache(ctx context.Context) error {
	result := r.client.EvalSha(ctx, r.cleanCacheSha, []string{"0"}, r.keyPrefix+":*")
	if result.Err() != nil {
		r.logger.CtxError(ctx, "[CleanCache] clean cache error: %v", result.Err())
		return result.Err()
	}
	return nil
}

func (r *Redis) BatchKeyExist(ctx context.Context, keys []string) (bool, error) {
	result := r.client.EvalSha(ctx, r.batchExistSha, keys)
	if result.Err() != nil {
		r.logger.CtxError(ctx, "[BatchKeyExist] eval script error: %v", result.Err())
		return false, result.Err()
	}
	return result.Bool()
}

func (r *Redis) KeyExists(ctx context.Context, key string) (bool, error) {
	result := r.client.Exists(ctx, key)
	if result.Err() != nil {
		r.logger.CtxError(ctx, "[KeyExists] exists error: %v", result.Err())
		return false, result.Err()
	}
	if result.Val() == 1 {
		return true, nil
	}
	return false, nil
}

func (r *Redis) GetValue(ctx context.Context, key string) (data string, err error) {
	data, err = r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		err = ErrCacheNotFound
	}
	return
}

func (r *Redis) BatchGetValues(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}
	result := r.client.MGet(ctx, keys...)
	if result.Err() != nil {
		r.logger.CtxError(ctx, "[BatchGetValues] mget error: %v", result.Err())
		return nil, result.Err()
	}
	slice := result.Val()
	// 确保返回的切片长度与keys长度一致，nil值用空字符串表示
	strs := make([]string, len(keys))
	for i, obj := range slice {
		if i >= len(keys) {
			break
		}
		if obj != nil {
			if str, ok := obj.(string); ok {
				strs[i] = str
			} else {
				r.logger.CtxError(ctx, "[BatchGetValues] unexpected type for key %s: %T", keys[i], obj)
				strs[i] = ""
			}
		}
		// obj == nil 时，strs[i] 保持为空字符串（零值）
	}
	return strs, nil
}

func (r *Redis) DeleteKeysWithPrefix(ctx context.Context, keyPrefix string) error {
	result := r.client.EvalSha(ctx, r.cleanCacheSha, []string{"0"}, keyPrefix+":*")
	return result.Err()
}

func (r *Redis) DeleteKey(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *Redis) BatchDeleteKeys(ctx context.Context, keys []string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *Redis) BatchSetKeys(ctx context.Context, kvs []util.Kv) error {
	if r.ttl == 0 {
		spreads := make([]interface{}, 0, len(kvs))
		for _, kv := range kvs {
			spreads = append(spreads, kv.Key)
			spreads = append(spreads, kv.Value)
		}
		return r.client.MSet(ctx, spreads...).Err()
	}
	_, err := r.client.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
		for _, kv := range kvs {
			result := pipeliner.Set(ctx, kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(r.ttl))*time.Millisecond)
			if result.Err() != nil {
				r.logger.CtxError(ctx, "[BatchSetKeys] set key %s error: %v", kv.Key, result.Err())
				return result.Err()
			}
		}
		return nil
	})
	return err
}

func (r *Redis) SetKey(ctx context.Context, kv util.Kv) error {
	return r.client.Set(ctx, kv.Key, kv.Value, time.Duration(util.RandFloatingInt64(r.ttl))*time.Millisecond).Err()
}
