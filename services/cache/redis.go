package cache

import (
	"context"
	"time"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/go-redis/redis/v8"
)

type RedisCacheBackend struct {
	endpoint string
	password string
	db       int
}

func NewRedisBackend(settings *model.CacheSettings) *RedisCacheBackend {
	return &RedisCacheBackend{
		endpoint: *settings.CacheEndpoint,
		password: "",
		db:       0, // default DB
	}
}

// ttlは毎回変更される。
func (b *RedisCacheBackend) Set(key string, value interface{}, expireSeconds int) error {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	err := rdb.Set(ctx, key, value, time.Duration(expireSeconds)*time.Second).Err()
	if err != nil {
		return err
	}

	return nil
}

func (b *RedisCacheBackend) Get(key string) (string, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.Get(ctx, key).Result()
}

func (b *RedisCacheBackend) Exists(key string) (int64, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.Exists(ctx, key).Result()
}

// ttlの変更はされ無い。
func (b *RedisCacheBackend) IncrBy(key string, count int) (int64, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.IncrBy(ctx, key, int64(count)).Result()
}
