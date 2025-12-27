package limiter

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBucket 基于Redis的令牌桶限流器
type RedisBucket struct {
	client     *redis.Client
	key        string
	rate       float64
	capacity   int64
	replenish  chan struct{}
	stop       chan struct{}
}

// NewRedisBucket 创建一个新的Redis令牌桶限流器
func NewRedisBucket(client *redis.Client, key string, rate float64, capacity int64) *RedisBucket {
	bucket := &RedisBucket{
		client:    client,
		key:       key,
		rate:      rate,
		capacity:  capacity,
		replenish: make(chan struct{}),
		stop:      make(chan struct{}),
	}

	// 初始化Lua脚本
	bucket.initLuaScripts()

	return bucket
}

// 初始化Lua脚本
func (b *RedisBucket) initLuaScripts() {
	// 定义获取令牌的Lua脚本
	allowScript := `
	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local tokens = tonumber(ARGV[4])
	local key = KEYS[1]
	local lastRefillTime = key .. ":last_refill"

	local last = tonumber(redis.call("get", lastRefillTime) or now)
	local delta = (now - last) / 1000 * rate
	local currentTokens = math.min(capacity, (tonumber(redis.call("get", key) or capacity) + delta))

	local allowed = 0
	if currentTokens >= tokens then
		allowed = 1
		currentTokens = currentTokens - tokens
	end

	redis.call("set", key, currentTokens)
	redis.call("set", lastRefillTime, now)
	redis.call("expire", key, 86400) -- 24小时过期
	redis.call("expire", lastRefillTime, 86400)

	return {allowed, currentTokens}
	`

	// 注册Lua脚本
	b.client.ScriptLoad(context.Background(), allowScript)

	// 定义获取多令牌的Lua脚本
	allowNScript := `
	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local tokens = tonumber(ARGV[4])
	local key = KEYS[1]
	local lastRefillTime = key .. ":last_refill"

	local last = tonumber(redis.call("get", lastRefillTime) or now)
	local delta = (now - last) / 1000 * rate
	local currentTokens = math.min(capacity, (tonumber(redis.call("get", key) or capacity) + delta))

	local allowed = tokens <= currentTokens and tokens or 0
	local remaining = currentTokens

	if allowed > 0 then
		remaining = currentTokens - allowed
	end

	redis.call("set", key, remaining)
	redis.call("set", lastRefillTime, now)
	redis.call("expire", key, 86400) -- 24小时过期
	redis.call("expire", lastRefillTime, 86400)

	return {allowed, remaining}
	`

	// 注册Lua脚本
	b.client.ScriptLoad(context.Background(), allowNScript)
}

// Allow 尝试获取1个令牌
func (b *RedisBucket) Allow(ctx context.Context) (bool, error) {
	allowed, _, err := b.AllowN(ctx, 1)
	return allowed, err
}

// AllowN 尝试获取指定数量的令牌
func (b *RedisBucket) AllowN(ctx context.Context, tokens int64) (bool, int64, error) {
	if tokens <= 0 {
		return false, 0, errors.New("tokens must be greater than 0")
	}

	// 定义获取多令牌的Lua脚本
	allowNScript := `
	local rate = tonumber(ARGV[1])
	local capacity = tonumber(ARGV[2])
	local now = tonumber(ARGV[3])
	local tokens = tonumber(ARGV[4])
	local key = KEYS[1]
	local lastRefillTime = key .. ":last_refill"

	local last = tonumber(redis.call("get", lastRefillTime) or now)
	local delta = (now - last) / 1000 * rate
	local currentTokens = math.min(capacity, (tonumber(redis.call("get", key) or capacity) + delta))

	local allowed = tokens <= currentTokens and tokens or 0
	local remaining = currentTokens

	if allowed > 0 then
		remaining = currentTokens - allowed
	end

	redis.call("set", key, remaining)
	redis.call("set", lastRefillTime, now)
	redis.call("expire", key, 86400) -- 24小时过期
	redis.call("expire", lastRefillTime, 86400)

	return {allowed, remaining}
	`

	now := time.Now().UnixNano() / int64(time.Millisecond)
	res, err := b.client.Eval(ctx, allowNScript, []string{b.key}, b.rate, b.capacity, now, tokens).Result()
	if err != nil {
		return false, 0, err
	}

	// 检查返回值类型
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, errors.New("invalid response from redis")
	}

	// 处理第一个返回值（allowed）
	var allowed int64
	if arr[0] != nil {
		switch v := arr[0].(type) {
		case int64:
			allowed = v
		case bool:
			if v {
				allowed = tokens
			}
		default:
			return false, 0, errors.New("unknown type for allowed")
		}
	}

	// 处理第二个返回值（remaining tokens）
	var remaining int64
	if arr[1] != nil {
		switch v := arr[1].(type) {
		case int64:
			remaining = v
		case float64:
			remaining = int64(v)
		default:
			return allowed > 0, 0, nil
		}
	}

	return allowed > 0, remaining, nil
}

// Close 关闭限流器
func (b *RedisBucket) Close() error {
	close(b.stop)
	return nil
}