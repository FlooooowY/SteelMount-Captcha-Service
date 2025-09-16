package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// IPBlocker handles IP blocking and suspicious activity detection
type IPBlocker struct {
	redis         *redis.Client
	mu            sync.RWMutex
	localBlocks   map[string]*BlockInfo
	failedAttempts map[string]*AttemptInfo
}

// BlockInfo represents information about a blocked IP
type BlockInfo struct {
	IP        string
	Reason    string
	BlockedAt time.Time
	ExpiresAt time.Time
	Attempts  int
}

// AttemptInfo represents failed attempt information
type AttemptInfo struct {
	IP        string
	Count     int
	FirstSeen time.Time
	LastSeen  time.Time
}

// NewIPBlocker creates a new IP blocker
func NewIPBlocker(redisClient *redis.Client) *IPBlocker {
	return &IPBlocker{
		redis:          redisClient,
		localBlocks:    make(map[string]*BlockInfo),
		failedAttempts: make(map[string]*AttemptInfo),
	}
}

// IsBlocked checks if an IP is currently blocked
func (ib *IPBlocker) IsBlocked(ctx context.Context, ip string) (bool, *BlockInfo, error) {
	// Check Redis first if available
	if ib.redis != nil {
		blocked, blockInfo, err := ib.checkRedisBlock(ctx, ip)
		if err == nil {
			return blocked, blockInfo, nil
		}
		// Fall back to local blocks if Redis fails
	}

	// Check local blocks
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	
	blockInfo, exists := ib.localBlocks[ip]
	if !exists {
		return false, nil, nil
	}
	
	// Check if block has expired
	if time.Now().After(blockInfo.ExpiresAt) {
		// Block expired, remove it
		delete(ib.localBlocks, ip)
		return false, nil, nil
	}
	
	return true, blockInfo, nil
}

// RecordFailedAttempt records a failed attempt for an IP
func (ib *IPBlocker) RecordFailedAttempt(ctx context.Context, ip string, reason string) error {
	// Record in Redis if available
	if ib.redis != nil {
		err := ib.recordRedisAttempt(ctx, ip, reason)
		if err == nil {
			return nil
		}
		// Fall back to local recording if Redis fails
	}

	// Record locally
	ib.mu.Lock()
	defer ib.mu.Unlock()
	
	now := time.Now()
	attemptInfo, exists := ib.failedAttempts[ip]
	
	if !exists {
		attemptInfo = &AttemptInfo{
			IP:        ip,
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
		ib.failedAttempts[ip] = attemptInfo
	} else {
		attemptInfo.Count++
		attemptInfo.LastSeen = now
	}
	
	// Check if IP should be blocked
	if attemptInfo.Count >= 5 { // Configurable threshold
		ib.blockIP(ip, reason, attemptInfo.Count)
	}
	
	return nil
}

// BlockIP manually blocks an IP address
func (ib *IPBlocker) BlockIP(ctx context.Context, ip string, reason string, duration time.Duration) error {
	blockInfo := &BlockInfo{
		IP:        ip,
		Reason:    reason,
		BlockedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Attempts:  0,
	}
	
	// Block in Redis if available
	if ib.redis != nil {
		err := ib.blockRedisIP(ctx, ip, blockInfo)
		if err == nil {
			return nil
		}
		// Fall back to local blocking if Redis fails
	}
	
	// Block locally
	ib.mu.Lock()
	defer ib.mu.Unlock()
	
	ib.localBlocks[ip] = blockInfo
	return nil
}

// UnblockIP removes a block from an IP address
func (ib *IPBlocker) UnblockIP(ctx context.Context, ip string) error {
	// Remove from Redis if available
	if ib.redis != nil {
		err := ib.unblockRedisIP(ctx, ip)
		if err == nil {
			return nil
		}
	}
	
	// Remove from local blocks
	ib.mu.Lock()
	defer ib.mu.Unlock()
	
	delete(ib.localBlocks, ip)
	delete(ib.failedAttempts, ip)
	return nil
}

// GetBlockedIPs returns a list of currently blocked IPs
func (ib *IPBlocker) GetBlockedIPs() []*BlockInfo {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	
	now := time.Now()
	var blockedIPs []*BlockInfo
	
	for _, blockInfo := range ib.localBlocks {
		if now.Before(blockInfo.ExpiresAt) {
			blockedIPs = append(blockedIPs, blockInfo)
		}
	}
	
	return blockedIPs
}

// CleanupExpiredBlocks removes expired blocks and attempts
func (ib *IPBlocker) CleanupExpiredBlocks() {
	ib.mu.Lock()
	defer ib.mu.Unlock()
	
	now := time.Now()
	
	// Clean up expired blocks
	for ip, blockInfo := range ib.localBlocks {
		if now.After(blockInfo.ExpiresAt) {
			delete(ib.localBlocks, ip)
		}
	}
	
	// Clean up old failed attempts (older than 1 hour)
	for ip, attemptInfo := range ib.failedAttempts {
		if now.Sub(attemptInfo.LastSeen) > time.Hour {
			delete(ib.failedAttempts, ip)
		}
	}
}

// checkRedisBlock checks if IP is blocked in Redis
func (ib *IPBlocker) checkRedisBlock(ctx context.Context, ip string) (bool, *BlockInfo, error) {
	key := fmt.Sprintf("blocked_ip:%s", ip)
	
	// Check if IP is blocked
	exists, err := ib.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}
	
	if exists == 0 {
		return false, nil, nil
	}
	
	// Get block information
	blockData, err := ib.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return false, nil, err
	}
	
	blockInfo := &BlockInfo{
		IP:        ip,
		Reason:    blockData["reason"],
		BlockedAt: time.Unix(0, 0), // Would need to parse from blockData
		ExpiresAt: time.Unix(0, 0), // Would need to parse from blockData
		Attempts:  0, // Would need to parse from blockData
	}
	
	return true, blockInfo, nil
}

// recordRedisAttempt records failed attempt in Redis
func (ib *IPBlocker) recordRedisAttempt(ctx context.Context, ip string, reason string) error {
	key := fmt.Sprintf("failed_attempts:%s", ip)
	
	// Increment counter
	count, err := ib.redis.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	
	// Set expiration (1 hour)
	ib.redis.Expire(ctx, key, time.Hour)
	
	// Check if should block
	if count >= 5 {
		blockInfo := &BlockInfo{
			IP:        ip,
			Reason:    reason,
			BlockedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Attempts:  int(count),
		}
		return ib.blockRedisIP(ctx, ip, blockInfo)
	}
	
	return nil
}

// blockRedisIP blocks IP in Redis
func (ib *IPBlocker) blockRedisIP(ctx context.Context, ip string, blockInfo *BlockInfo) error {
	key := fmt.Sprintf("blocked_ip:%s", ip)
	
	// Store block information
	blockData := map[string]interface{}{
		"reason":     blockInfo.Reason,
		"blocked_at": blockInfo.BlockedAt.Unix(),
		"expires_at": blockInfo.ExpiresAt.Unix(),
		"attempts":   blockInfo.Attempts,
	}
	
	return ib.redis.HMSet(ctx, key, blockData).Err()
}

// unblockRedisIP removes IP block from Redis
func (ib *IPBlocker) unblockRedisIP(ctx context.Context, ip string) error {
	key := fmt.Sprintf("blocked_ip:%s", ip)
	return ib.redis.Del(ctx, key).Err()
}

// blockIP blocks IP locally
func (ib *IPBlocker) blockIP(ip string, reason string, attempts int) {
	blockInfo := &BlockInfo{
		IP:        ip,
		Reason:    reason,
		BlockedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Attempts:  attempts,
	}
	
	ib.localBlocks[ip] = blockInfo
}

// GetStats returns IP blocker statistics
func (ib *IPBlocker) GetStats() map[string]interface{} {
	ib.mu.RLock()
	defer ib.mu.RUnlock()
	
	return map[string]interface{}{
		"blocked_ips":      len(ib.localBlocks),
		"failed_attempts":  len(ib.failedAttempts),
		"redis_available":  ib.redis != nil,
	}
}
