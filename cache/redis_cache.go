package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// 基于 Redis 的缓存实现。
type RedisCache struct {
	client *redis.Client
}

// 创建基于 Redis 的缓存。
func NewRedisCache(addr string, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCache{client: client}
}

// 使用已有 Redis 客户端创建缓存。
func NewRedisCacheWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// 将值写入 Redis。
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration, randomExpiration ...bool) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	expiration = addRandomExpiration(expiration, randomExpiration...)
	return r.client.Set(ctx, key, data, expiration).Err()
}

// 将缓存值读取到目标对象。
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrKeyNotFound
		}
		return err
	}
	return json.Unmarshal(data, dest)
}

// 删除缓存值。
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// 判断键是否存在于 Redis。
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

// 关闭 Redis 客户端。
func (r *RedisCache) Close() error {
	return r.client.Close()
}
