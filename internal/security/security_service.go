package security

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// SecurityService provides comprehensive security features
type SecurityService struct {
	rateLimiter *RateLimiter
	ipBlocker   *IPBlocker
	botDetector *BotDetector
	config      *SecurityConfig
	mu          sync.RWMutex
	stats       *SecurityStats
}

// SecurityStats tracks security metrics
type SecurityStats struct {
	TotalRequests       int64
	BlockedRequests     int64
	RateLimitedRequests int64
	BotDetections       int64
	IPBlocks            int64
	StartTime           time.Time
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	RateLimitConfig    RateLimitConfig
	IPBlockingConfig   IPBlockingConfig
	BotDetectionConfig BotDetectionConfig
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool
	RequestsPerMinute int
	BurstSize         int
	Window            time.Duration
}

// IPBlockingConfig represents IP blocking configuration
type IPBlockingConfig struct {
	Enabled           bool
	MaxFailedAttempts int
	BlockDuration     time.Duration
	CleanupInterval   time.Duration
}

// BotDetectionConfig represents bot detection configuration
type BotDetectionConfig struct {
	Enabled         bool
	MinBotScore     float64
	HighBotScore    float64
	CleanupInterval time.Duration
}

// NewSecurityService creates a new security service
func NewSecurityService(redisClient *redis.Client, config *SecurityConfig) *SecurityService {
	return &SecurityService{
		rateLimiter: NewRateLimiter(redisClient),
		ipBlocker:   NewIPBlockerWithConfig(redisClient, config.IPBlockingConfig.MaxFailedAttempts),
		botDetector: NewBotDetector(),
		config:      config,
		stats: &SecurityStats{
			StartTime: time.Now(),
		},
	}
}

// CheckRequest performs comprehensive security checks on a request
func (ss *SecurityService) CheckRequest(ctx context.Context, ip string, userAgent string, path string, responseTime time.Duration, isError bool) (*SecurityResult, error) {
	ss.mu.Lock()
	ss.stats.TotalRequests++
	ss.mu.Unlock()

	result := &SecurityResult{
		IP:        ip,
		Timestamp: time.Now(),
		Allowed:   true,
		Reasons:   []string{},
	}

	// Check if IP is blocked
	blocked, blockInfo, err := ss.ipBlocker.IsBlocked(ctx, ip)
	if err != nil {
		return nil, fmt.Errorf("failed to check IP block: %w", err)
	}

	if blocked {
		result.Allowed = false
		result.Reasons = append(result.Reasons, fmt.Sprintf("IP blocked: %s", blockInfo.Reason))
		ss.mu.Lock()
		ss.stats.BlockedRequests++
		ss.mu.Unlock()
		return result, nil
	}

	// Check rate limiting
	allowed, err := ss.rateLimiter.Allow(ctx, ip, ss.config.RateLimitConfig.RequestsPerMinute, time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if !allowed {
		result.Allowed = false
		result.Reasons = append(result.Reasons, "Rate limit exceeded")
		ss.mu.Lock()
		ss.stats.RateLimitedRequests++
		ss.mu.Unlock()
		return result, nil
	}

	// Analyze for bot behavior
	botScore, err := ss.botDetector.AnalyzeRequest(ctx, ip, userAgent, path, responseTime, isError)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze bot behavior: %w", err)
	}

	// Check bot score
	if botScore.Score > 0.7 { // High bot probability
		result.Allowed = false
		result.Reasons = append(result.Reasons, fmt.Sprintf("Bot detected (score: %.2f)", botScore.Score))
		result.Reasons = append(result.Reasons, botScore.Reasons...)

		// Record failed attempt
		ss.ipBlocker.RecordFailedAttempt(ctx, ip, "Bot behavior detected")

		ss.mu.Lock()
		ss.stats.BotDetections++
		ss.mu.Unlock()
		return result, nil
	} else if botScore.Score > 0.4 { // Medium bot probability
		result.Reasons = append(result.Reasons, fmt.Sprintf("Suspicious behavior (score: %.2f)", botScore.Score))
		result.Reasons = append(result.Reasons, botScore.Reasons...)
	}

	// Record failed attempt if there was an error
	if isError {
		ss.ipBlocker.RecordFailedAttempt(ctx, ip, "Request error")
		
		// Check if IP was blocked due to failed attempts
		blocked, blockInfo, err := ss.ipBlocker.IsBlocked(ctx, ip)
		if err != nil {
			return nil, fmt.Errorf("failed to check IP block after failed attempt: %w", err)
		}
		
		if blocked {
			result.Allowed = false
			result.Reasons = append(result.Reasons, fmt.Sprintf("IP blocked: %s", blockInfo.Reason))
			ss.mu.Lock()
			ss.stats.BlockedRequests++
			ss.mu.Unlock()
			return result, nil
		}
	}

	return result, nil
}

// BlockIP manually blocks an IP address
func (ss *SecurityService) BlockIP(ctx context.Context, ip string, reason string, duration time.Duration) error {
	err := ss.ipBlocker.BlockIP(ctx, ip, reason, duration)
	if err != nil {
		return fmt.Errorf("failed to block IP: %w", err)
	}

	ss.mu.Lock()
	ss.stats.IPBlocks++
	ss.mu.Unlock()

	return nil
}

// UnblockIP removes a block from an IP address
func (ss *SecurityService) UnblockIP(ctx context.Context, ip string) error {
	return ss.ipBlocker.UnblockIP(ctx, ip)
}

// GetBlockedIPs returns a list of currently blocked IPs
func (ss *SecurityService) GetBlockedIPs() []*BlockInfo {
	return ss.ipBlocker.GetBlockedIPs()
}

// GetStats returns security service statistics
func (ss *SecurityService) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	uptime := time.Since(ss.stats.StartTime)

	stats := map[string]interface{}{
		"total_requests":        ss.stats.TotalRequests,
		"blocked_requests":      ss.stats.BlockedRequests,
		"rate_limited_requests": ss.stats.RateLimitedRequests,
		"bot_detections":        ss.stats.BotDetections,
		"ip_blocks":             ss.stats.IPBlocks,
		"uptime_seconds":        uptime.Seconds(),
		"request_rate":          float64(ss.stats.TotalRequests) / uptime.Seconds(),
	}

	// Add component stats
	rateLimiterStats := ss.rateLimiter.GetStats()
	ipBlockerStats := ss.ipBlocker.GetStats()
	botDetectorStats := ss.botDetector.GetStats()

	stats["rate_limiter"] = rateLimiterStats
	stats["ip_blocker"] = ipBlockerStats
	stats["bot_detector"] = botDetectorStats

	return stats
}

// Cleanup performs maintenance tasks
func (ss *SecurityService) Cleanup() {
	// Cleanup rate limiter
	ss.rateLimiter.CleanupExpiredLimits()

	// Cleanup IP blocker
	ss.ipBlocker.CleanupExpiredBlocks()

	// Cleanup bot detector
	ss.botDetector.CleanupExpiredPatterns()
}

// StartCleanupRoutine starts a background cleanup routine
func (ss *SecurityService) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ss.Cleanup()
		}
	}
}

// SecurityResult represents the result of a security check
type SecurityResult struct {
	IP        string
	Allowed   bool
	Reasons   []string
	Timestamp time.Time
}

// IsValidIP checks if an IP address is valid
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ExtractIPFromRequest extracts IP from request context
func ExtractIPFromRequest(ctx context.Context) string {
	// This would typically extract IP from gRPC context
	// For now, return a placeholder
	return "127.0.0.1"
}

// ExtractUserAgentFromRequest extracts user agent from request context
func ExtractUserAgentFromRequest(ctx context.Context) string {
	// This would typically extract user agent from gRPC context
	// For now, return a placeholder
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
}
