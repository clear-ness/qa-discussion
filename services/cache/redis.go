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
// expire 0 はttl無し、と言う意味。
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

func (b *RedisCacheBackend) HSet(key string, values map[string]interface{}) (int64, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.HSet(ctx, key, values).Result()
}

func (b *RedisCacheBackend) HGetAll(key string) (map[string]string, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.HGetAll(ctx, key).Result()
}

// 重複を許さない文字列集合
func (b *RedisCacheBackend) SAdd(key string, members []string) (int64, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.SAdd(ctx, key, members).Result()
}

func (b *RedisCacheBackend) SMembers(key string) ([]string, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.SMembers(ctx, key).Result()
}

func (b *RedisCacheBackend) Del(keys []string) (int64, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.Del(ctx, keys...).Result()
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

func (b *RedisCacheBackend) FlushAll() (string, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     b.endpoint,
		Password: b.password,
		DB:       b.db,
	})

	return rdb.FlushAll(ctx).Result()
}
