package cache

import (
	"math/rand/v2"
	"time"
)

// 按需给过期时间增加随机偏移，避免大量缓存同时失效。
func addRandomExpiration(expiration time.Duration, randomExpiration ...bool) time.Duration {
	if len(randomExpiration) == 0 || !randomExpiration[0] || expiration <= 0 {
		return expiration
	}

	maxRandomExpiration := expiration / 3
	if maxRandomExpiration <= 0 {
		return expiration
	}

	return expiration + time.Duration(rand.Int64N(int64(maxRandomExpiration)+1))
}
