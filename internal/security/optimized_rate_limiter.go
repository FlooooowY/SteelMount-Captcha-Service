package security

import (
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// OptimizedRateLimiter provides high-performance rate limiting with sliding window
type OptimizedRateLimiter struct {
	redis       *redis.Client
	mu          sync.RWMutex
	localLimits map[string]*OptimizedLocalLimit
	cleanupTicker *time.Ticker
	stopCleanup chan struct{}
}

// OptimizedLocalLimit represents an optimized local rate limit
type OptimizedLocalLimit struct {
	Count     int
	LastReset time.Time
	Window    time.Duration
	// Sliding window for more accurate rate limiting
	requests []time.Time
	mu       sync.RWMutex
}

// NewOptimizedRateLimiter creates a new optimized rate limiter
func NewOptimizedRateLimiter(redisClient *redis.Client) *OptimizedRateLimiter {
	limiter := &OptimizedRateLimiter{
		redis:       redisClient,
		localLimits: make(map[string]*OptimizedLocalLimit),
		stopCleanup: make(chan struct{}),
	}
	
	// Start cleanup routine
	limiter.cleanupTicker = time.NewTicker(30 * time.Second)
	go limiter.cleanupRoutine()
	
	return limiter
}

// Allow checks if a request is allowed using sliding window algorithm
func (orl *OptimizedRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Try Redis first if available
	if orl.redis != nil {
		allowed, err := orl.checkRedisLimit(ctx, key, limit, window)
		if err == nil {
			return allowed, nil
		}
		// Fall back to local limits if Redis fails
	}

	// Use optimized local rate limiting
	return orl.checkOptimizedLocalLimit(key, limit, window), nil
}

// checkOptimizedLocalLimit checks rate limit using sliding window
func (orl *OptimizedRateLimiter) checkOptimizedLocalLimit(key string, limit int, window time.Duration) bool {
	orl.mu.Lock()
	defer orl.mu.Unlock()
	
	now := time.Now()
	localLimit, exists := orl.localLimits[key]
	
	if !exists {
		localLimit = &OptimizedLocalLimit{
			Count:     0,
			LastReset: now,
			Window:    window,
			requests:  make([]time.Time, 0, limit*2), // Pre-allocate with capacity
		}
		orl.localLimits[key] = localLimit
	}
	
	// Use sliding window algorithm
	return localLimit.checkSlidingWindow(now, limit, window)
}

// checkSlidingWindow implements sliding window rate limiting
func (ol *OptimizedLocalLimit) checkSlidingWindow(now time.Time, limit int, window time.Duration) bool {
	ol.mu.Lock()
	defer ol.mu.Unlock()
	
	// Remove old requests outside the window
	cutoff := now.Add(-window)
	validRequests := ol.requests[:0] // Reset slice but keep capacity
	
	for _, reqTime := range ol.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	ol.requests = validRequests
	
	// Check if we're under the limit
	if len(ol.requests) < limit {
		ol.requests = append(ol.requests, now)
		ol.Count = len(ol.requests)
		ol.LastReset = now
		return true
	}
	
	return false
}

// checkRedisLimit checks rate limit using Redis with sliding window
func (orl *OptimizedRateLimiter) checkRedisLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Use Redis sorted set for sliding window
	now := time.Now()
	cutoff := now.Add(-window).Unix()
	
	pipe := orl.redis.Pipeline()
	
	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", string(cutoff))
	
	// Count current entries
	count := pipe.ZCard(ctx, key)
	
	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.Unix()),
		Member: now.UnixNano(),
	})
	
	// Set expiration
	pipe.Expire(ctx, key, window)
	
	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}
	
	// Check if under limit
	currentCount, err := count.Result()
	if err != nil {
		return false, err
	}
	
	return currentCount < int64(limit), nil
}

// cleanupRoutine periodically cleans up expired limits
func (orl *OptimizedRateLimiter) cleanupRoutine() {
	for {
		select {
		case <-orl.cleanupTicker.C:
			orl.cleanupExpiredLimits()
		case <-orl.stopCleanup:
			return
		}
	}
}

// cleanupExpiredLimits removes expired local limits
func (orl *OptimizedRateLimiter) cleanupExpiredLimits() {
	orl.mu.Lock()
	defer orl.mu.Unlock()
	
	now := time.Now()
	for key, limit := range orl.localLimits {
		if now.Sub(limit.LastReset) >= limit.Window*2 { // Keep for 2x window for safety
			delete(orl.localLimits, key)
		}
	}
}

// GetStats returns rate limiter statistics
func (orl *OptimizedRateLimiter) GetStats() map[string]interface{} {
	orl.mu.RLock()
	defer orl.mu.RUnlock()
	
	return map[string]interface{}{
		"active_limits":  len(orl.localLimits),
		"redis_available": orl.redis != nil,
		"cleanup_running": orl.cleanupTicker != nil,
	}
}

// Close stops the cleanup routine
func (orl *OptimizedRateLimiter) Close() {
	if orl.cleanupTicker != nil {
		orl.cleanupTicker.Stop()
	}
	close(orl.stopCleanup)
}
