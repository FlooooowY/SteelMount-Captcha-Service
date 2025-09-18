package unit

import (
	"context"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

func TestBotDetector_AnalyzeRequest(t *testing.T) {
	detector := security.NewBotDetector()
	ctx := context.Background()

	tests := []struct {
		name         string
		ip           string
		userAgent    string
		path         string
		responseTime time.Duration
		isError      bool
		expectBot    bool
	}{
		{
			name:         "normal user",
			ip:           "192.168.1.1",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			path:         "/api/captcha",
			responseTime: time.Millisecond * 100,
			isError:      false,
			expectBot:    false,
		},
		{
			name:         "bot user agent",
			ip:           "192.168.1.2",
			userAgent:    "bot/crawler/1.0",
			path:         "/api/captcha",
			responseTime: time.Millisecond * 50,
			isError:      false,
			expectBot:    true,
		},
		{
			name:         "headless browser",
			ip:           "192.168.1.3",
			userAgent:    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.124",
			path:         "/api/captcha",
			responseTime: time.Millisecond * 30,
			isError:      false,
			expectBot:    true,
		},
		{
			name:         "suspicious path",
			ip:           "192.168.1.4",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			path:         "/admin/config",
			responseTime: time.Millisecond * 200,
			isError:      false,
			expectBot:    true,
		},
		{
			name:         "high error rate",
			ip:           "192.168.1.5",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			path:         "/api/captcha",
			responseTime: time.Millisecond * 100,
			isError:      true,
			expectBot:    false, // Single error shouldn't trigger bot detection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := detector.AnalyzeRequest(ctx, tt.ip, tt.userAgent, tt.path, tt.responseTime, tt.isError)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			isBot := score.Score > 0.7
			if isBot != tt.expectBot {
				t.Errorf("Expected bot=%v, got bot=%v (score: %.2f)", tt.expectBot, isBot, score.Score)
			}
		})
	}
}

func TestBotDetector_RequestPatterns(t *testing.T) {
	detector := security.NewBotDetector()
	ctx := context.Background()

	// Simulate bot behavior with high frequency requests
	ip := "192.168.1.100"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Send many requests quickly (bot-like behavior)
	for i := 0; i < 20; i++ {
		score, err := detector.AnalyzeRequest(ctx, ip, userAgent, path, time.Millisecond*10, false)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// After several requests, should detect bot behavior
		if i > 10 && score.Score < 0.4 {
			t.Errorf("Expected high bot score after %d requests, got %.2f", i, score.Score)
		}
	}
}

func TestBotDetector_Cleanup(t *testing.T) {
	detector := security.NewBotDetector()
	ctx := context.Background()

	// Create some request patterns
	_, _ = detector.AnalyzeRequest(ctx, "192.168.1.1", "Mozilla/5.0", "/api/captcha", time.Millisecond*100, false)
	_, _ = detector.AnalyzeRequest(ctx, "192.168.1.2", "Mozilla/5.0", "/api/captcha", time.Millisecond*100, false)

	// Get initial stats
	initialStats := detector.GetStats()
	initialTrackedIPs := initialStats["tracked_ips"].(int)

	if initialTrackedIPs == 0 {
		t.Errorf("Expected tracked IPs > 0, got %d", initialTrackedIPs)
	}

	// Cleanup expired patterns
	detector.CleanupExpiredPatterns()

	// Stats should still be the same since patterns are recent
	finalStats := detector.GetStats()
	finalTrackedIPs := finalStats["tracked_ips"].(int)

	if finalTrackedIPs != initialTrackedIPs {
		t.Errorf("Expected tracked IPs to remain %d after cleanup, got %d", initialTrackedIPs, finalTrackedIPs)
	}
}

func TestBotDetector_Performance(t *testing.T) {
	detector := security.NewBotDetector()
	ctx := context.Background()

	// Test performance with many requests
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		ip := "192.168.1.1"
		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
		path := "/api/captcha"

		_, err := detector.AnalyzeRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
		if err != nil {
			t.Errorf("Error: %v", err)
		}
	}

	duration := time.Since(start)
	rps := float64(iterations) / duration.Seconds()

	t.Logf("Processed %d bot detection requests in %v (RPS: %.2f)", iterations, duration, rps)

	// Should be able to process at least 500 RPS
	if rps < 500 {
		t.Errorf("RPS too low: %.2f, expected at least 500", rps)
	}
}
