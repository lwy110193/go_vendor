package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// TestRedisBucketAllow 测试Allow方法
func TestRedisBucketAllow(t *testing.T) {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 清理测试数据
	ctx := context.Background()
	client.Del(ctx, "ratelimit:token:bucket:test_key")

	// 创建限流器配置
	config := &Config{
		Rate:       10, // 每秒10个令牌
		Burst:      5,  // 最大5个令牌
		Expiration: time.Hour,
	}

	// 创建限流器
	limiter := NewRedisBucket(client, config)

	// 测试允许通过
	allowed, _, err := limiter.Allow(ctx, "test_key")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// 快速连续请求，消耗所有令牌
	for i := 0; i < 4; i++ {
		allowed, _, err = limiter.Allow(ctx, "test_key")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// 令牌应该用完了
	allowed, _, err = limiter.Allow(ctx, "test_key")
	assert.NoError(t, err)
	assert.False(t, allowed)

	// 等待令牌生成
	time.Sleep(200 * time.Millisecond) // 应该生成2个令牌
	allowed, _, err = limiter.Allow(ctx, "test_key")
	assert.NoError(t, err)
	assert.True(t, allowed)
}

// TestRedisBucketAllowN 测试AllowN方法
func TestRedisBucketAllowN(t *testing.T) {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 清理测试数据
	ctx := context.Background()
	client.Del(ctx, "ratelimit:token:bucket:test_key_n")

	// 创建限流器配置
	config := &Config{
		Rate:       10, // 每秒10个令牌
		Burst:      10, // 最大10个令牌
		Expiration: time.Hour,
	}

	// 创建限流器
	limiter := NewRedisBucket(client, config)

	// 测试一次性获取3个令牌
	allowed, tokens, err := limiter.AllowN(ctx, "test_key_n", 3)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(7), tokens) // 初始10个令牌，消耗3个，剩余7个

	// 测试获取超过剩余令牌数
	allowed, tokens, err = limiter.AllowN(ctx, "test_key_n", 8)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(7), tokens) // 令牌数不变

	// 测试获取0个令牌
	allowed, tokens, err = limiter.AllowN(ctx, "test_key_n", 0)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

// TestRedisBucketExpiration 测试令牌桶过期
func TestRedisBucketExpiration(t *testing.T) {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 清理测试数据
	ctx := context.Background()
	testKey := "ratelimit:token:bucket:test_expiration"
	client.Del(ctx, testKey)

	// 创建限流器配置，设置短过期时间
	config := &Config{
		Rate:       10,
		Burst:      5,
		Expiration: 2 * time.Second, // 2秒过期
	}

	// 创建限流器
	limiter := NewRedisBucket(client, config)

	// 消耗一些令牌
	for i := 0; i < 3; i++ {
		allowed, _, err := limiter.Allow(ctx, "test_expiration")
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	// 等待过期
	time.Sleep(2 * time.Second)

	// 验证键已过期
	exists, err := client.Exists(ctx, testKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), exists)

	// 再次请求应该重置令牌桶
	allowed, tokens, err := limiter.Allow(ctx, "test_expiration")
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(4), tokens) // 重置后应有5个令牌，消耗1个，剩余4个
}
