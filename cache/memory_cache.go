package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// MemoryCache 基于内存的缓存实现
type MemoryCache struct {
	// 存储缓存项的map
	items map[string]*memoryItem
	// 读写锁，保证并发安全
	mutex sync.RWMutex
	// 清理过期项的定时器
	cleanupTicker *time.Ticker
	// 停止清理的通道
	stopChan chan struct{}
}

// memoryItem 内存缓存项
type memoryItem struct {
	// 缓存值，已序列化
	value []byte
	// 过期时间
	expiration time.Time
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items:    make(map[string]*memoryItem),
		stopChan: make(chan struct{}),
	}

	// 启动清理过期项的后台协程
	cache.startCleanupRoutine()

	return cache
}

// startCleanupRoutine 启动清理过期项的后台协程
func (m *MemoryCache) startCleanupRoutine() {
	// 每5分钟清理一次过期项
	m.cleanupTicker = time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <- m.cleanupTicker.C:
				m.deleteExpired()
			case <- m.stopChan:
				m.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// deleteExpired 删除所有过期的缓存项
func (m *MemoryCache) deleteExpired() {
	now := time.Now()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, item := range m.items {
		if !item.expiration.IsZero() && now.After(item.expiration) {
			delete(m.items, key)
		}
	}
}

// Set 设置缓存
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 序列化数据
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// 计算过期时间
	var expiry time.Time
	if expiration > 0 {
		expiry = time.Now().Add(expiration)
	}

	m.mutex.Lock()
	m.items[key] = &memoryItem{
		value:      data,
		expiration: expiry,
	}
	m.mutex.Unlock()

	return nil
}

// Get 获取缓存
func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mutex.RLock()
	item, found := m.items[key]
	m.mutex.RUnlock()

	if !found {
		return ErrKeyNotFound
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		// 在获取时异步删除过期项
		go m.Delete(ctx, key)
		return ErrKeyNotFound
	}

	// 反序列化数据
	return json.Unmarshal(item.value, dest)
}

// Delete 删除缓存
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	m.mutex.Lock()
	delete(m.items, key)
	m.mutex.Unlock()

	return nil
}

// Exists 检查缓存是否存在
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	m.mutex.RLock()
	item, found := m.items[key]
	m.mutex.RUnlock()

	if !found {
		return false, nil
	}

	// 检查是否过期
	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		// 在检查时异步删除过期项
		go m.Delete(ctx, key)
		return false, nil
	}

	return true, nil
}

// Close 关闭缓存，停止清理协程
func (m *MemoryCache) Close() error {
	close(m.stopChan)
	m.mutex.Lock()
	m.items = make(map[string]*memoryItem) // 清空所有缓存项
	m.mutex.Unlock()
	return nil
}

// Size 获取当前缓存项数量
func (m *MemoryCache) Size() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.items)
}

// Clear 清空所有缓存项
func (m *MemoryCache) Clear() {
	m.mutex.Lock()
	m.items = make(map[string]*memoryItem)
	m.mutex.Unlock()
}
