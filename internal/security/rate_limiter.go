package security

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter handles rate limiting for requests
type RateLimiter struct {
	redis       *redis.Client
	mu          sync.RWMutex
	localLimits map[string]*LocalLimit
}

// LocalLimit represents a local rate limit
type LocalLimit struct {
	Count     int
	LastReset time.Time
	Window    time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:       redisClient,
		localLimits: make(map[string]*LocalLimit),
	}
}

// Allow checks if a request is allowed based on rate limits
func (rl *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Try Redis first if available
	if rl.redis != nil {
		allowed, err := rl.checkRedisLimit(ctx, key, limit, window)
		if err == nil {
			return allowed, nil
		}
		// Fall back to local limits if Redis fails
	}

	// Use local rate limiting as fallback
	return rl.checkLocalLimit(key, limit, window), nil
}

// checkRedisLimit checks rate limit using Redis
func (rl *RateLimiter) checkRedisLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	pipe := rl.redis.Pipeline()
	
	// Increment counter
	incr := pipe.Incr(ctx, key)
	
	// Set expiration if this is the first request
	pipe.Expire(ctx, key, window)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	
	// Check if limit exceeded
	count, err := incr.Result()
	if err != nil {
		return false, err
	}
	
	return count <= int64(limit), nil
}

// checkLocalLimit checks rate limit using local memory
func (rl *RateLimiter) checkLocalLimit(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	localLimit, exists := rl.localLimits[key]
	
	if !exists || now.Sub(localLimit.LastReset) >= window {
		// Reset or create new limit
		rl.localLimits[key] = &LocalLimit{
			Count:     1,
			LastReset: now,
			Window:    window,
		}
		return true
	}
	
	// Check if limit exceeded
	if localLimit.Count >= limit {
		return false
	}
	
	// Increment counter
	localLimit.Count++
	return true
}

// CleanupExpiredLimits removes expired local limits
func (rl *RateLimiter) CleanupExpiredLimits() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	for key, limit := range rl.localLimits {
		if now.Sub(limit.LastReset) >= limit.Window {
			delete(rl.localLimits, key)
		}
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	return map[string]interface{}{
		"active_limits": len(rl.localLimits),
		"redis_available": rl.redis != nil,
	}
}
