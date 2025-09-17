package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/captcha"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

// TestCaptchaGenerationRPS tests captcha generation performance
func TestCaptchaGenerationRPS(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))
	engine.RegisterCaptcha("click", captcha.NewClickGenerator(400, 300, 2, 5))
	engine.RegisterCaptcha("swipe", captcha.NewSwipeGenerator(400, 300, 1, 3))

	testCases := []struct {
		name        string
		captchaType string
		iterations  int
		targetRPS   float64
	}{
		{
			name:        "drag_drop_100_rps",
			captchaType: "drag_drop",
			iterations:  1000,
			targetRPS:   100,
		},
		{
			name:        "click_100_rps",
			captchaType: "click",
			iterations:  1000,
			targetRPS:   100,
		},
		{
			name:        "swipe_100_rps",
			captchaType: "swipe",
			iterations:  1000,
			targetRPS:   100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()

			for i := 0; i < tc.iterations; i++ {
				_, _, err := engine.GenerateChallenge(tc.captchaType, 50)
				if err != nil {
					t.Errorf("Error generating challenge: %v", err)
				}
			}

			duration := time.Since(start)
			rps := float64(tc.iterations) / duration.Seconds()

			t.Logf("Generated %d %s captchas in %v (RPS: %.2f)",
				tc.iterations, tc.captchaType, duration, rps)

			if rps < tc.targetRPS {
				t.Errorf("RPS too low: %.2f, expected at least %.2f", rps, tc.targetRPS)
			}
		})
	}
}

// TestConcurrentCaptchaGeneration tests concurrent captcha generation
func TestConcurrentCaptchaGeneration(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))

	concurrency := 50
	iterationsPerGoroutine := 20
	totalIterations := concurrency * iterationsPerGoroutine

	start := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				_, _, err := engine.GenerateChallenge("drag_drop", 50)
				if err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error generating challenge: %v", err)
	}

	duration := time.Since(start)
	rps := float64(totalIterations) / duration.Seconds()

	t.Logf("Generated %d captchas concurrently in %v (RPS: %.2f)",
		totalIterations, duration, rps)

	// Should be able to generate at least 200 RPS with concurrency
	if rps < 200 {
		t.Errorf("Concurrent RPS too low: %.2f, expected at least 200", rps)
	}
}

// TestMemoryUsage tests memory usage during high load
func TestMemoryUsage(t *testing.T) {
	engine := captcha.NewEngine()
	engine.RegisterCaptcha("drag_drop", captcha.NewDragDropGenerator(400, 300, 3, 8))

	// Measure initial memory
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Generate many captchas
	iterations := 10000

	for i := 0; i < iterations; i++ {
		_, _, err := engine.GenerateChallenge("drag_drop", 50)
		if err != nil {
			t.Errorf("Error generating challenge: %v", err)
		}

		// Force GC every 1000 iterations
		if i%1000 == 0 {
			runtime.GC()
		}
	}

	// Force final GC and measure memory
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	memoryUsed := m2.Alloc - m1.Alloc
	memoryPerCaptcha := float64(memoryUsed) / float64(iterations)

	t.Logf("Memory used: %d bytes (%d KB)", memoryUsed, memoryUsed/1024)
	t.Logf("Memory per captcha: %.2f bytes", memoryPerCaptcha)

	// Memory per captcha should be reasonable (less than 1KB)
	if memoryPerCaptcha > 1024 {
		t.Errorf("Memory per captcha too high: %.2f bytes, expected less than 1024", memoryPerCaptcha)
	}
}

// TestSecurityPerformance tests security system performance
func TestSecurityPerformance(t *testing.T) {
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
	iterations := 1000

	start := time.Now()

	for i := 0; i < iterations; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i%255)
		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
		path := "/api/captcha"

		_, err := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*100, false)
		if err != nil {
			t.Errorf("Error checking request: %v", err)
		}
	}

	duration := time.Since(start)
	rps := float64(iterations) / duration.Seconds()

	t.Logf("Processed %d security checks in %v (RPS: %.2f)", iterations, duration, rps)

	// Should be able to process at least 500 RPS
	if rps < 500 {
		t.Errorf("Security RPS too low: %.2f, expected at least 500", rps)
	}
}

// TestConcurrentSecurityChecks tests concurrent security checks
func TestConcurrentSecurityChecks(t *testing.T) {
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

	concurrency := 20
	iterationsPerGoroutine := 50
	totalIterations := concurrency * iterationsPerGoroutine

	start := time.Now()

	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < iterationsPerGoroutine; j++ {
				ip := fmt.Sprintf("192.168.%d.%d", goroutineID%255, j%255)
				userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
				path := "/api/captcha"

				_, err := securityService.CheckRequest(context.Background(), ip, userAgent, path, time.Millisecond*100, false)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Error checking request: %v", err)
	}

	duration := time.Since(start)
	rps := float64(totalIterations) / duration.Seconds()

	t.Logf("Processed %d security checks concurrently in %v (RPS: %.2f)",
		totalIterations, duration, rps)

	// Should be able to process at least 1000 RPS with concurrency
	if rps < 1000 {
		t.Errorf("Concurrent security RPS too low: %.2f, expected at least 1000", rps)
	}
}
