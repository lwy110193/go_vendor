package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache 定义缓存接口
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
}

// ErrKeyNotFound 当键不存在时返回的错误
var ErrKeyNotFound = errors.New("key not found")

// RedisCache 基于Redis的缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(addr string, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCache{client: client}
}

// NewRedisClient 创建Redis客户端
func NewRedisCacheWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// Set 设置缓存
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 将值序列化为JSON
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrKeyNotFound
		}
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists 检查键是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	res, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

// Close 关闭Redis连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}
