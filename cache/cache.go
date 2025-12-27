package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
)

// 定义错误
var (
	// ErrKeyNotFound 键不存在错误
	ErrKeyNotFound = errors.New("cache: key not found")
)

// Cache 缓存接口
type Cache interface {
	// Set 设置缓存，支持任意类型的值
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	// Get 获取缓存，返回的结果需要进行类型断言
	Get(ctx context.Context, key string, dest interface{}) error
	// Delete 删除缓存
	Delete(ctx context.Context, key string) error
	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)
	// Close 关闭缓存连接
	Close() error
}

// RedisCache 基于Redis的缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

// Set 设置缓存
// key: 缓存键
// value: 缓存值，可以是任意类型
// expiration: 过期时间
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 序列化数据
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// 调用Redis的Set命令
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Get 获取缓存
// key: 缓存键
// dest: 用于接收数据的指针，需要与存储时的数据类型匹配
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	// 获取数据
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrKeyNotFound
		}
		return err
	}

	// 反序列化数据到目标对象
	return json.Unmarshal([]byte(data), dest)
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists 检查缓存是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// Close 关闭Redis连接
func (r *RedisCache) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// NewRedisClient 创建默认的Redis客户端
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}

// NewRedisCacheWithClient 创建Redis缓存实例，使用已有的Redis客户端
func NewRedisCacheWithClient(client *redis.Client) *RedisCache {
	return &RedisCache{
		client: client,
	}
}
