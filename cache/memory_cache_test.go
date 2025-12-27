package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试内存缓存的基本操作
func TestMemoryCacheBasicOperations(t *testing.T) {
	// 创建内存缓存实例
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	// 测试数据
	key := "test_key"
	value := "test_value"

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
	assert.Equal(t, ErrKeyNotFound, err)

	// 测试Size方法
	assert.Equal(t, 0, cache.Size())
}

// 测试缓存过期功能
func TestMemoryCacheExpiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	key := "expiration_test_key"
	value := "expiration_test_value"

	// 设置一个2秒后过期的缓存
	err := cache.Set(ctx, key, value, 2*time.Second)
	assert.NoError(t, err)

	// 立即检查缓存是否存在
	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 获取缓存并验证值
	var result string
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	// 等待3秒，确保缓存已过期
	time.Sleep(3 * time.Second)

	// 检查缓存是否已过期（通过Exists）
	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 检查缓存是否已过期（通过Get）
	err = cache.Get(ctx, key, &result)
	assert.Error(t, err)
	assert.Equal(t, ErrKeyNotFound, err)
}

// 测试缓存复杂数据类型
func TestMemoryCacheComplexTypes(t *testing.T) {
	cache := NewMemoryCache()
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

	// 设置切片缓存
	err = cache.Set(ctx, sliceKey, sliceValue, time.Hour)
	assert.NoError(t, err)

	// 获取切片缓存
	var sliceResult []int
	err = cache.Get(ctx, sliceKey, &sliceResult)
	assert.NoError(t, err)
	assert.Equal(t, sliceValue, sliceResult)

	// 测试映射类型
	mapKey := "map_type_key"
	mapValue := map[string]interface{}{
		"name":   "Test",
		"age":    25,
		"scores": []float64{95.5, 98.0},
	}

	// 设置映射缓存
	err = cache.Set(ctx, mapKey, mapValue, time.Hour)
	assert.NoError(t, err)

	// 获取映射缓存
	var mapResult map[string]interface{}
	err = cache.Get(ctx, mapKey, &mapResult)
	assert.NoError(t, err)
	assert.Equal(t, mapValue["name"], mapResult["name"])
	assert.Equal(t, float64(25), mapResult["age"])
}

// 测试上下文取消
func TestMemoryCacheContextCancel(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	// 测试Set
	err := cache.Set(ctx, "test_key", "test_value", time.Hour)
	assert.Error(t, err)

	// 测试Get
	var result string
	err = cache.Get(ctx, "test_key", &result)
	assert.Error(t, err)

	// 测试Delete
	err = cache.Delete(ctx, "test_key")
	assert.Error(t, err)

	// 测试Exists
	_, err = cache.Exists(ctx, "test_key")
	assert.Error(t, err)
}

// 测试Clear方法
func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	// 设置多个缓存项
	err := cache.Set(ctx, "key1", "value1", time.Hour)
	assert.NoError(t, err)
	err = cache.Set(ctx, "key2", "value2", time.Hour)
	assert.NoError(t, err)
	err = cache.Set(ctx, "key3", "value3", time.Hour)
	assert.NoError(t, err)

	// 验证大小
	assert.Equal(t, 3, cache.Size())

	// 清空缓存
	cache.Clear()

	// 验证大小为0
	assert.Equal(t, 0, cache.Size())

	// 验证缓存项已不存在
	exists, err := cache.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// 测试并发安全性
func TestMemoryCacheConcurrency(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()
	ctx := context.Background()

	// 并发操作的数量
	opCount := 1000
	goroutineCount := 10

	// 等待组，确保所有goroutine完成
	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	// 错误通道
	errChan := make(chan error, goroutineCount*opCount)

	// 启动多个goroutine进行并发操作
	for g := 0; g < goroutineCount; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < opCount; i++ {
				key := fmt.Sprintf("concurrency_key_%d_%d", goroutineID, i)
				value := fmt.Sprintf("concurrency_value_%d_%d", goroutineID, i)

				// 设置缓存
				if err := cache.Set(ctx, key, value, time.Hour); err != nil {
					errChan <- err
					continue
				}

				// 检查缓存是否存在
				if exists, err := cache.Exists(ctx, key); err != nil || !exists {
					errChan <- fmt.Errorf("key %s should exist", key)
					continue
				}

				// 获取缓存
				var result string
				if err := cache.Get(ctx, key, &result); err != nil || result != value {
					errChan <- fmt.Errorf("get error: %v or value mismatch", err)
					continue
				}

				// 删除缓存
				if err := cache.Delete(ctx, key); err != nil {
					errChan <- err
					continue
				}

				// 检查缓存是否已删除
				if exists, err := cache.Exists(ctx, key); err != nil || exists {
					errChan <- fmt.Errorf("key %s should not exist", key)
				}
			}
		}(g)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	errors := make([]error, 0)
	for err := range errChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors, "并发操作应该没有错误")
}
