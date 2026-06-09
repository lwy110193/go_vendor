package cache

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newTestRedisCache(t *testing.T) *RedisCache {
	t.Helper()

	addr := os.Getenv("REDIS_CACHE_ADDR")
	if addr == "" {
		t.Skip("set REDIS_CACHE_ADDR to run Redis cache integration tests")
	}

	db := 0
	if value := os.Getenv("REDIS_CACHE_DB"); value != "" {
		parsed, err := strconv.Atoi(value)
		assert.NoError(t, err)
		db = parsed
	}

	return NewRedisCache(addr, os.Getenv("REDIS_CACHE_PASSWORD"), db)
}

func TestRedisCacheBasicOperations(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	_ = cache.Delete(ctx, key)

	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	var result string
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = cache.Get(ctx, key, &result)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestRedisCacheExpiration(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()
	key := "expiration_test_key"
	value := "expiration_test_value"

	_ = cache.Delete(ctx, key)

	err := cache.Set(ctx, key, value, 2*time.Second)
	assert.NoError(t, err)

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	time.Sleep(3 * time.Second)

	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestRedisCacheComplexTypes(t *testing.T) {
	cache := newTestRedisCache(t)
	defer cache.Close()

	ctx := context.Background()

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

	_ = cache.Delete(ctx, key)

	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	var result TestStruct
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	sliceKey := "slice_type_key"
	sliceValue := []int{1, 2, 3, 4, 5}

	_ = cache.Delete(ctx, sliceKey)

	err = cache.Set(ctx, sliceKey, sliceValue, time.Hour)
	assert.NoError(t, err)

	var sliceResult []int
	err = cache.Get(ctx, sliceKey, &sliceResult)
	assert.NoError(t, err)
	assert.Equal(t, sliceValue, sliceResult)
}

func TestMemoryCacheBasicOperations(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	var result string
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = cache.Get(ctx, key, &result)
	assert.ErrorIs(t, err, ErrKeyNotFound)
	assert.Equal(t, 0, cache.Size())
}

func TestMemoryCacheExpiration(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	key := "expiration_test_key"
	value := "expiration_test_value"

	err := cache.Set(ctx, key, value, 2*time.Second)
	assert.NoError(t, err)

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	var result string
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	time.Sleep(3 * time.Second)

	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = cache.Get(ctx, key, &result)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestMemoryCacheComplexTypes(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

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

	err := cache.Set(ctx, key, value, time.Hour)
	assert.NoError(t, err)

	var result TestStruct
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	sliceKey := "slice_type_key"
	sliceValue := []int{1, 2, 3, 4, 5}

	err = cache.Set(ctx, sliceKey, sliceValue, time.Hour)
	assert.NoError(t, err)

	var sliceResult []int
	err = cache.Get(ctx, sliceKey, &sliceResult)
	assert.NoError(t, err)
	assert.Equal(t, sliceValue, sliceResult)

	mapKey := "map_type_key"
	mapValue := map[string]interface{}{
		"name":   "Test",
		"age":    25,
		"scores": []float64{95.5, 98.0},
	}

	err = cache.Set(ctx, mapKey, mapValue, time.Hour)
	assert.NoError(t, err)

	var mapResult map[string]interface{}
	err = cache.Get(ctx, mapKey, &mapResult)
	assert.NoError(t, err)
	assert.Equal(t, mapValue["name"], mapResult["name"])
	assert.Equal(t, float64(25), mapResult["age"])
}

func TestMemoryCacheContextCancel(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cache.Set(ctx, "test_key", "test_value", time.Hour)
	assert.Error(t, err)

	var result string
	err = cache.Get(ctx, "test_key", &result)
	assert.Error(t, err)

	err = cache.Delete(ctx, "test_key")
	assert.Error(t, err)

	_, err = cache.Exists(ctx, "test_key")
	assert.Error(t, err)
}

func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()

	err := cache.Set(ctx, "key1", "value1", time.Hour)
	assert.NoError(t, err)
	err = cache.Set(ctx, "key2", "value2", time.Hour)
	assert.NoError(t, err)
	err = cache.Set(ctx, "key3", "value3", time.Hour)
	assert.NoError(t, err)

	assert.Equal(t, 3, cache.Size())

	cache.Clear()

	assert.Equal(t, 0, cache.Size())

	exists, err := cache.Exists(ctx, "key1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestMemoryCacheConcurrency(t *testing.T) {
	cache := NewMemoryCache()
	defer cache.Close()

	ctx := context.Background()
	opCount := 1000
	goroutineCount := 10

	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	errChan := make(chan error, goroutineCount*opCount)

	for g := 0; g < goroutineCount; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			for i := 0; i < opCount; i++ {
				key := fmt.Sprintf("concurrency_key_%d_%d", goroutineID, i)
				value := fmt.Sprintf("concurrency_value_%d_%d", goroutineID, i)

				if err := cache.Set(ctx, key, value, time.Hour); err != nil {
					errChan <- err
					continue
				}

				if exists, err := cache.Exists(ctx, key); err != nil || !exists {
					errChan <- fmt.Errorf("key %s should exist", key)
					continue
				}

				var result string
				if err := cache.Get(ctx, key, &result); err != nil || result != value {
					errChan <- fmt.Errorf("get error: %v or value mismatch", err)
					continue
				}

				if err := cache.Delete(ctx, key); err != nil {
					errChan <- err
					continue
				}

				if exists, err := cache.Exists(ctx, key); err != nil || exists {
					errChan <- fmt.Errorf("key %s should not exist", key)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errChan)

	errors := make([]error, 0)
	for err := range errChan {
		errors = append(errors, err)
	}

	assert.Empty(t, errors)
}

func TestMemoryCacheCloseIdempotent(t *testing.T) {
	cache := NewMemoryCache()

	assert.NoError(t, cache.Close())
	assert.NoError(t, cache.Close())

	err := cache.Set(context.Background(), "key", "value", time.Hour)
	assert.ErrorIs(t, err, ErrCacheClosed)
}

func TestMemoryCacheExpiredReadDoesNotDeleteNewValue(t *testing.T) {
	cache := NewMemoryCacheWithCleanupInterval(0)
	defer cache.Close()

	ctx := context.Background()
	key := "reused_key"

	assert.NoError(t, cache.Set(ctx, key, "old", time.Nanosecond))
	time.Sleep(time.Millisecond)

	var result string
	err := cache.Get(ctx, key, &result)
	assert.ErrorIs(t, err, ErrKeyNotFound)

	assert.NoError(t, cache.Set(ctx, key, "new", time.Hour))

	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, "new", result)
}

func TestAddRandomExpiration(t *testing.T) {
	expiration := 3 * time.Second

	assert.Equal(t, expiration, addRandomExpiration(expiration))
	assert.Equal(t, expiration, addRandomExpiration(expiration, false))
	assert.Equal(t, time.Duration(0), addRandomExpiration(0, true))
	assert.Equal(t, -time.Second, addRandomExpiration(-time.Second, true))

	for i := 0; i < 100; i++ {
		got := addRandomExpiration(expiration, true)
		assert.GreaterOrEqual(t, got, expiration)
		assert.LessOrEqual(t, got, expiration+expiration/3)
	}
}

func TestMemoryCacheSetWithRandomExpiration(t *testing.T) {
	cache := NewMemoryCacheWithCleanupInterval(0)
	defer cache.Close()

	ctx := context.Background()
	key := "random_expiration_key"
	expiration := 3 * time.Second
	start := time.Now()

	assert.NoError(t, cache.Set(ctx, key, "value", expiration, true))

	cache.mutex.RLock()
	item := cache.items[key]
	cache.mutex.RUnlock()

	assert.NotNil(t, item)
	assert.False(t, item.expiration.IsZero())
	assert.GreaterOrEqual(t, item.expiration.Sub(start), expiration)
	assert.LessOrEqual(t, item.expiration.Sub(start), expiration+expiration/3+100*time.Millisecond)
}
