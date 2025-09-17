package unit

import (
	"context"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

func TestRateLimiter_LocalLimits(t *testing.T) {
	// Test without Redis (local-only mode)
	limiter := security.NewRateLimiter(nil)
	
	tests := []struct {
		name     string
		key      string
		limit    int
		window   time.Duration
		requests int
		expected []bool
	}{
		{
			name:     "within limit",
			key:      "test1",
			limit:    5,
			window:   time.Minute,
			requests: 3,
			expected: []bool{true, true, true},
		},
		{
			name:     "at limit",
			key:      "test2",
			limit:    3,
			window:   time.Minute,
			requests: 3,
			expected: []bool{true, true, true},
		},
		{
			name:     "over limit",
			key:      "test3",
			limit:    2,
			window:   time.Minute,
			requests: 4,
			expected: []bool{true, true, false, false},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			for i := 0; i < tt.requests; i++ {
				allowed, err := limiter.Allow(ctx, tt.key, tt.limit, tt.window)
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				
				expected := tt.expected[i]
				if allowed != expected {
					t.Errorf("Request %d: expected %v, got %v", i, expected, allowed)
				}
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := security.NewRateLimiter(nil)
	ctx := context.Background()
	
	// Create some limits
	limiter.Allow(ctx, "key1", 5, time.Millisecond*100)
	limiter.Allow(ctx, "key2", 5, time.Millisecond*100)
	
	// Wait for expiration
	time.Sleep(time.Millisecond * 150)
	
	// Cleanup should remove expired limits
	limiter.CleanupExpiredLimits()
	
	// Check stats
	stats := limiter.GetStats()
	activeLimits := stats["active_limits"].(int)
	
	if activeLimits != 0 {
		t.Errorf("Expected 0 active limits after cleanup, got %d", activeLimits)
	}
}

func TestRateLimiter_Performance(t *testing.T) {
	limiter := security.NewRateLimiter(nil)
	ctx := context.Background()
	
	// Test performance with many requests
	iterations := 1000
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		key := "perf_test"
		_, err := limiter.Allow(ctx, key, 1000, time.Minute)
		if err != nil {
			t.Errorf("Error: %v", err)
		}
	}
	
	duration := time.Since(start)
	rps := float64(iterations) / duration.Seconds()
	
	t.Logf("Processed %d rate limit checks in %v (RPS: %.2f)", iterations, duration, rps)
	
	// Should be able to process at least 1000 RPS
	if rps < 1000 {
		t.Errorf("RPS too low: %.2f, expected at least 1000", rps)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := security.NewRateLimiter(nil)
	ctx := context.Background()
	
	// Test concurrent access
	concurrency := 10
	iterations := 100
	
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				key := "concurrent_test"
				_, err := limiter.Allow(ctx, key, 1000, time.Minute)
				if err != nil {
					t.Errorf("Error: %v", err)
				}
			}
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}
