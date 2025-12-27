package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试Redis缓存的基本操作
func TestRedisCacheBasicOperations(t *testing.T) {
	// 创建缓存实例
	cache := NewRedisCache("192.168.3.42:6379", "redis_MK8zA6", 0)
	defer cache.Close()

	ctx := context.Background()

	// 测试数据
	key := "test_key"
	value := "test_value"

	// 清理可能存在的测试数据
	cache.Delete(ctx, key)

	// 测试设置缓存
	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	// 测试检查缓存是否存在
	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试获取缓存
	var result string
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// 测试删除缓存
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// 测试缓存是否已删除
	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 测试获取不存在的缓存
	err = cache.Get(ctx, key, &result)
	assert.Error(t, err)
}

// 测试缓存过期功能
func TestRedisCacheExpiration(t *testing.T) {
	cache := NewRedisCache("192.168.3.42:6379", "redis_MK8zA6", 0)
	defer cache.Close()

	ctx := context.Background()

	key := "expiration_test_key"
	value := "expiration_test_value"

	// 清理可能存在的测试数据
	cache.Delete(ctx, key)

	// 设置一个2秒后过期的缓存
	err := cache.Set(ctx, key, value, 2*time.Second)
	assert.NoError(t, err)

	// 立即检查缓存是否存在
	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 等待3秒，确保缓存已过期
	time.Sleep(3 * time.Second)

	// 检查缓存是否已过期
	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// 测试缓存复杂数据类型
func TestRedisCacheComplexTypes(t *testing.T) {
	cache := NewRedisCache("192.168.3.42:6379", "redis_MK8zA6", 0)
	defer cache.Close()

	ctx := context.Background()

	// 测试结构体类型
	type TestStruct struct {
		Name  string
		Age   int
		Score float64
	}

	key := "complex_type_key"
	value := TestStruct{
		Name:  "Test",
		Age:   25,
		Score: 95.5,
	}

	// 清理可能存在的测试数据
	cache.Delete(ctx, key)

	// 设置结构体缓存
	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	// 获取结构体缓存
	var result TestStruct
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// 测试切片类型
	sliceKey := "slice_type_key"
	sliceValue := []int{1, 2, 3, 4, 5}

	// 清理可能存在的测试数据
	cache.Delete(ctx, sliceKey)

	// 设置切片缓存
	err = cache.Set(ctx, sliceKey, sliceValue, time.Hour)
	assert.NoError(t, err)

	// 获取切片缓存
	var sliceResult []int
	err = cache.Get(ctx, sliceKey, &sliceResult)
	assert.NoError(t, err)
	assert.Equal(t, sliceValue, sliceResult)
}
