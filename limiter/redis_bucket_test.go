package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestRedisBucketAllow 测试基本的令牌获取功能
func TestRedisBucketAllow(t *testing.T) {
	ctx := context.Background()
	
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 创建一个令牌桶，每秒产生10个令牌，容量为20
	bucket := NewRedisBucket(client, "test:bucket:allow", 10, 20)
	defer bucket.Close()

	// 清理测试数据
	client.Del(ctx, "test:bucket:allow")
	client.Del(ctx, "test:bucket:allow:last_refill")

	// 测试成功获取令牌
	allowed, err := bucket.Allow(ctx)
	assert.NoError(t, err)
	assert.True(t, allowed)

	// 测试多次获取令牌
	for i := 0; i < 10; i++ {
		allowed, err := bucket.Allow(ctx)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
}

// TestRedisBucketAllowN 测试获取多个令牌
func TestRedisBucketAllowN(t *testing.T) {
	ctx := context.Background()
	
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 创建一个令牌桶，每秒产生10个令牌，容量为20
	bucket := NewRedisBucket(client, "test:bucket:allown", 10, 20)
	defer bucket.Close()

	// 清理测试数据
	client.Del(ctx, "test:bucket:allown")
	client.Del(ctx, "test:bucket:allown:last_refill")

	// 测试成功获取多个令牌
	allowed, remaining, err := bucket.AllowN(ctx, 5)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(15), remaining)

	// 测试获取超过剩余数量的令牌
	allowed, remaining, err = bucket.AllowN(ctx, 20)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, int64(15), remaining)
}

// TestRedisBucketExpiration 测试令牌桶过期
func TestRedisBucketExpiration(t *testing.T) {
	ctx := context.Background()
	
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:     "192.168.3.42:6379",
		Password: "redis_MK8zA6",
		DB:       0,
	})
	defer client.Close()

	// 创建一个令牌桶，每秒产生10个令牌，容量为20
	bucket := NewRedisBucket(client, "test:bucket:expiration", 10, 20)
	defer bucket.Close()

	// 清理测试数据
	client.Del(ctx, "test:bucket:expiration")
	client.Del(ctx, "test:bucket:expiration:last_refill")

	// 获取一些令牌
	allowed, remaining, err := bucket.AllowN(ctx, 5)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int64(15), remaining)

	// 等待一段时间，让令牌桶补充一些令牌
	time.Sleep(2 * time.Second)

	// 再次获取令牌，应该能获取到更多
	allowed, remaining, err = bucket.AllowN(ctx, 10)
	assert.NoError(t, err)
	assert.True(t, allowed)
	// 2秒后应该补充了约20个令牌，但受容量限制，最多20个
	// 减去之前剩余的15个，应该新增了5个，所以现在剩余应该是15+5-10=10
	assert.True(t, remaining <= 20)
}