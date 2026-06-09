package cache

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

const defaultCleanupInterval = 5 * time.Minute

// 基于内存的缓存实现。
type MemoryCache struct {
	items map[string]*memoryItem

	mutex         sync.RWMutex
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	closeOnce     sync.Once
	closed        bool
}

type memoryItem struct {
	value      []byte
	expiration time.Time
}

// 使用默认清理间隔创建内存缓存。
func NewMemoryCache() *MemoryCache {
	return NewMemoryCacheWithCleanupInterval(defaultCleanupInterval)
}

// 使用自定义清理间隔创建内存缓存。
// 清理间隔小于等于 0 时禁用后台清理。
func NewMemoryCacheWithCleanupInterval(cleanupInterval time.Duration) *MemoryCache {
	cache := &MemoryCache{
		items:    make(map[string]*memoryItem),
		stopChan: make(chan struct{}),
	}

	if cleanupInterval > 0 {
		cache.startCleanupRoutine(cleanupInterval)
	}

	return cache
}

func (m *MemoryCache) startCleanupRoutine(interval time.Duration) {
	m.cleanupTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.deleteExpired()
			case <-m.stopChan:
				m.cleanupTicker.Stop()
				return
			}
		}
	}()
}

func (m *MemoryCache) deleteExpired() {
	now := time.Now()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, item := range m.items {
		if item.expired(now) {
			delete(m.items, key)
		}
	}
}

func (m *MemoryCache) deleteExpiredKey(key string, item *memoryItem, now time.Time) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	current, found := m.items[key]
	if found && current == item && current.expired(now) {
		delete(m.items, key)
	}
}

func (i *memoryItem) expired(now time.Time) bool {
	return i != nil && !i.expiration.IsZero() && !now.Before(i.expiration)
}

// 写入缓存值。
func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration, randomExpiration ...bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	var expiry time.Time
	expiration = addRandomExpiration(expiration, randomExpiration...)
	if expiration > 0 {
		expiry = time.Now().Add(expiration)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return ErrCacheClosed
	}

	m.items[key] = &memoryItem{
		value:      data,
		expiration: expiry,
	}
	return nil
}

// 将缓存值读取到目标对象。
func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now()

	m.mutex.RLock()
	item, found := m.items[key]
	closed := m.closed
	m.mutex.RUnlock()

	if closed {
		return ErrCacheClosed
	}
	if !found {
		return ErrKeyNotFound
	}
	if item.expired(now) {
		m.deleteExpiredKey(key, item, now)
		return ErrKeyNotFound
	}

	return json.Unmarshal(item.value, dest)
}

// 删除缓存值。
func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return ErrCacheClosed
	}

	delete(m.items, key)
	return nil
}

// 判断未过期的键是否存在。
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	now := time.Now()

	m.mutex.RLock()
	item, found := m.items[key]
	closed := m.closed
	m.mutex.RUnlock()

	if closed {
		return false, ErrCacheClosed
	}
	if !found {
		return false, nil
	}
	if item.expired(now) {
		m.deleteExpiredKey(key, item, now)
		return false, nil
	}

	return true, nil
}

// 停止后台清理并清空缓存值。
func (m *MemoryCache) Close() error {
	m.closeOnce.Do(func() {
		if m.cleanupTicker != nil {
			close(m.stopChan)
		}

		m.mutex.Lock()
		m.closed = true
		m.items = make(map[string]*memoryItem)
		m.mutex.Unlock()
	})

	return nil
}

// 返回未过期缓存项数量。
func (m *MemoryCache) Size() int {
	m.deleteExpired()

	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.items)
}

// 清空所有缓存值。
func (m *MemoryCache) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.closed {
		return
	}

	m.items = make(map[string]*memoryItem)
}
