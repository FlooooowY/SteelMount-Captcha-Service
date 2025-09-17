package security

import (
	"context"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

func TestSecurityService_IPBlocking(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 3,
			BlockDuration:     time.Hour,
			CleanupInterval:   time.Hour,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         true,
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	})

	ctx := context.Background()
	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Test normal request
	result, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Normal request should be allowed")
	}

	// Simulate failed attempts
	for i := 0; i < 3; i++ {
		result, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, true)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should still be allowed during failed attempts
		if !result.Allowed {
			t.Errorf("Request should be allowed during failed attempts")
		}
	}

	// One more failed attempt should trigger blocking
	result, err = securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, true)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Allowed {
		t.Errorf("Request should be blocked after max failed attempts")
	}

	// Check if IP is in blocked list
	blockedIPs := securityService.GetBlockedIPs()
	if len(blockedIPs) == 0 {
		t.Errorf("IP should be in blocked list")
	}

	found := false
	for _, blockedIP := range blockedIPs {
		if blockedIP.IP == ip {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("IP %s should be in blocked list", ip)
	}
}

func TestSecurityService_RateLimiting(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 5, // Very low limit for testing
			BurstSize:         2,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           false, // Disable IP blocking for this test
			MaxFailedAttempts: 5,
			BlockDuration:     time.Hour,
			CleanupInterval:   time.Hour,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         false, // Disable bot detection for this test
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	})

	ctx := context.Background()
	ip := "192.168.1.200"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// First few requests should be allowed
	for i := 0; i < 5; i++ {
		result, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !result.Allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Next request should be rate limited
	result, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Allowed {
		t.Errorf("Request should be rate limited")
	}

	// Check reasons
	found := false
	for _, reason := range result.Reasons {
		if reason == "Rate limit exceeded" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Rate limit reason not found in: %v", result.Reasons)
	}
}

func TestSecurityService_BotDetection(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           false, // Disable IP blocking for this test
			MaxFailedAttempts: 5,
			BlockDuration:     time.Hour,
			CleanupInterval:   time.Hour,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         true,
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	})

	ctx := context.Background()
	ip := "192.168.1.300"
	path := "/api/captcha"

	// Test bot user agent
	botUserAgent := "bot/crawler/1.0"
	result, err := securityService.CheckRequest(ctx, ip, botUserAgent, path, time.Millisecond*10, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Allowed {
		t.Errorf("Bot request should be blocked")
	}

	// Check reasons
	found := false
	for _, reason := range result.Reasons {
		if reason == "Bot detected" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Bot detection reason not found in: %v", result.Reasons)
	}

	// Test normal user agent
	normalUserAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	result, err = securityService.CheckRequest(ctx, "192.168.1.301", normalUserAgent, path, time.Millisecond*100, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Normal request should be allowed")
	}
}

func TestSecurityService_ManualIPBlocking(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 5,
			BlockDuration:     time.Hour,
			CleanupInterval:   time.Hour,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         true,
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	})

	ctx := context.Background()
	ip := "192.168.1.400"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Manually block IP
	err := securityService.BlockIP(ctx, ip, "Manual test block", time.Hour)
	if err != nil {
		t.Errorf("Failed to block IP: %v", err)
	}

	// Test that blocked IP is rejected
	result, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Allowed {
		t.Errorf("Blocked IP should be rejected")
	}

	// Check reasons
	found := false
	for _, reason := range result.Reasons {
		if reason == "IP blocked: Manual test block" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("IP block reason not found in: %v", result.Reasons)
	}

	// Unblock IP
	err = securityService.UnblockIP(ctx, ip)
	if err != nil {
		t.Errorf("Failed to unblock IP: %v", err)
	}

	// Test that unblocked IP is allowed
	result, err = securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Errorf("Unblocked IP should be allowed")
	}
}

func TestSecurityService_Stats(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 5,
			BlockDuration:     time.Hour,
			CleanupInterval:   time.Hour,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         true,
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	})

	ctx := context.Background()
	ip := "192.168.1.500"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Make some requests
	for i := 0; i < 10; i++ {
		securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
	}

	// Get stats
	stats := securityService.GetStats()

	// Check that stats are populated
	if stats["total_requests"].(int64) == 0 {
		t.Errorf("Total requests should be > 0")
	}

	if stats["uptime_seconds"].(float64) == 0 {
		t.Errorf("Uptime should be > 0")
	}

	t.Logf("Security stats: %+v", stats)
}
