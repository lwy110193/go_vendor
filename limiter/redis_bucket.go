package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisBucket 基于Redis的令牌桶限流器
type RedisBucket struct {
	client *redis.Client
	config *Config
	// Lua脚本，用于原子性操作令牌桶
	takeScript  *redis.Script
	takeNScript *redis.Script
}

// NewRedisBucket 创建Redis令牌桶限流器
func NewRedisBucket(client *redis.Client, config *Config) *RedisBucket {
	if config == nil {
		config = NewDefaultConfig()
	}

	// 初始化Lua脚本
	takeScript := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local ttl = tonumber(ARGV[4])

		-- 获取当前令牌桶状态
		local current = redis.call('HMGET', key, 'last_refill_time', 'tokens')
		local lastRefillTime = tonumber(current[1]) or now
		local tokens = tonumber(current[2]) or burst

		-- 计算新的令牌数
		local elapsed = now - lastRefillTime
		local newTokens = tokens + elapsed * rate / 1000.0

		-- 限制最大令牌数
		if newTokens > burst then
			newTokens = burst
		end

		-- 尝试消耗1个令牌
		local allowed = newTokens >= 1
		if allowed then
			newTokens = newTokens - 1
		end

		-- 更新令牌桶状态
		redis.call('HMSET', key, 'last_refill_time', now, 'tokens', newTokens)
		redis.call('EXPIRE', key, ttl)

		return {allowed, newTokens}
	`)

	takeNScript := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local ttl = tonumber(ARGV[4])
		local n = tonumber(ARGV[5])

		-- 获取当前令牌桶状态
		local current = redis.call('HMGET', key, 'last_refill_time', 'tokens')
		local lastRefillTime = tonumber(current[1]) or now
		local tokens = tonumber(current[2]) or burst

		-- 计算新的令牌数
		local elapsed = now - lastRefillTime
		local newTokens = tokens + elapsed * rate / 1000.0

		-- 限制最大令牌数
		if newTokens > burst then
			newTokens = burst
		end

		-- 尝试消耗n个令牌
		local allowed = newTokens >= n
		if allowed then
			newTokens = newTokens - n
		end

		-- 更新令牌桶状态
		redis.call('HMSET', key, 'last_refill_time', now, 'tokens', newTokens)
		redis.call('EXPIRE', key, ttl)

		return {allowed, newTokens}
	`)

	return &RedisBucket{
		client:      client,
		config:      config,
		takeScript:  takeScript,
		takeNScript: takeNScript,
	}
}

// Allow 判断是否允许通过
func (r *RedisBucket) Allow(ctx context.Context, key string) (bool, int64, error) {
	return r.AllowN(ctx, key, 1)
}

// AllowN 判断是否允许通过N个请求
func (r *RedisBucket) AllowN(ctx context.Context, key string, n int64) (bool, int64, error) {
	if n <= 0 {
		return true, 0, nil
	}

	// 构建Redis key
	redisKey := fmt.Sprintf("ratelimit:token:bucket:%s", key)
	now := time.Now().UnixMilli()

	// 执行Lua脚本
	result, err := r.takeNScript.Run(ctx, r.client, []string{redisKey},
		r.config.Rate,
		r.config.Burst,
		now,
		int(r.config.Expiration.Seconds()),
		n,
	).Result()

	if err != nil {
		return false, 0, err
	}

	// 类型断言为[]interface{}
	arr, ok := result.([]interface{})
	if !ok {
		return false, 0, fmt.Errorf("invalid result type: %T", result)
	}

	// 处理allowed的类型，可能是int64或nil
	allowed := false
	if arr[0] != nil {
		switch v := arr[0].(type) {
		case int64:
			allowed = v > 0
		case bool:
			allowed = v
		}
	}

	// 正确处理tokens的类型，可能是int64、float64或nil
	var tokens int64
	if arr[1] != nil {
		switch v := arr[1].(type) {
		case int64:
			tokens = v
		case float64:
			tokens = int64(v)
		default:
			return false, 0, fmt.Errorf("invalid tokens type: %T", v)
		}
	}

	return allowed, tokens, nil
}

// Close 关闭限流器连接
func (r *RedisBucket) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
