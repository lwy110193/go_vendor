package limiter

import (
	"context"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 判断是否允许通过
	// key: 限流键
	// 返回值: 是否允许通过，剩余令牌数
	Allow(ctx context.Context, key string) (bool, int64, error)

	// AllowN 判断是否允许通过N个请求
	// key: 限流键
	// n: 请求数量
	// 返回值: 是否允许通过，剩余令牌数
	AllowN(ctx context.Context, key string, n int64) (bool, int64, error)

	// Close 关闭限流器连接
	Close() error
}

// Config 限流器配置
type Config struct {
	// Rate: 每秒生成的令牌数
	Rate int64
	// Burst: 最大令牌数
	Burst int64
	// Expiration: 令牌桶过期时间，默认1小时
	Expiration time.Duration
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	return &Config{
		Rate:       10,        // 每秒10个令牌
		Burst:      20,        // 最多20个令牌
		Expiration: time.Hour, // 1小时过期
	}
}
