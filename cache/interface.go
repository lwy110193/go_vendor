package cache

import (
	"context"
	"errors"
	"time"
)

// 定义通用缓存操作。
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration, randomExpiration ...bool) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Close() error
}

// 表示键不存在或已经过期。
var ErrKeyNotFound = errors.New("key not found")

// 表示缓存已经关闭。
var ErrCacheClosed = errors.New("cache closed")
