package security

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
)

// TestDDoSAttack tests DDoS attack scenario
func TestDDoSAttack(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 100, // Allow 100 requests per minute
			BurstSize:         10,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 10,
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
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Simulate DDoS attack from multiple IPs
	attackIPs := []string{
		"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5",
		"10.0.0.6", "10.0.0.7", "10.0.0.8", "10.0.0.9", "10.0.0.10",
	}

	var wg sync.WaitGroup
	attackResults := make([][]bool, len(attackIPs))

	// Launch concurrent attacks from different IPs
	for i, ip := range attackIPs {
		wg.Add(1)
		go func(ipIndex int, attackIP string) {
			defer wg.Done()
			
			results := make([]bool, 200) // 200 requests per IP
			for j := 0; j < 200; j++ {
				result, err := securityService.CheckRequest(ctx, attackIP, userAgent, path, time.Millisecond*1, false)
				if err != nil {
					t.Errorf("Unexpected error from IP %s: %v", attackIP, err)
					return
				}
				results[j] = result.Allowed
				
				// Small delay to simulate realistic attack
				time.Sleep(time.Millisecond * 5)
			}
			attackResults[ipIndex] = results
		}(i, ip)
	}

	wg.Wait()

	// Analyze results
	totalRequests := 0
	totalBlocked := 0
	
	for i, results := range attackResults {
		allowed := 0
		blocked := 0
		for _, isAllowed := range results {
			totalRequests++
			if isAllowed {
				allowed++
			} else {
				blocked++
				totalBlocked++
			}
		}
		t.Logf("IP %s: %d allowed, %d blocked", attackIPs[i], allowed, blocked)
	}

	// Verify that rate limiting is working
	blockingPercentage := float64(totalBlocked) / float64(totalRequests) * 100
	t.Logf("Total requests: %d, blocked: %d (%.1f%%)", totalRequests, totalBlocked, blockingPercentage)

	if blockingPercentage < 50.0 {
		t.Errorf("Expected at least 50%% of requests to be blocked during DDoS attack, got %.1f%%", blockingPercentage)
	}
}

// TestBruteForceAttack tests brute force attack scenario
func TestBruteForceAttack(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 5, // Block after 5 failed attempts
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
	attackerIP := "192.168.100.1"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Simulate brute force attack with many failed attempts
	failedAttempts := 0
	for i := 0; i < 20; i++ {
		result, err := securityService.CheckRequest(ctx, attackerIP, userAgent, path, time.Millisecond*10, true) // Mark as failed
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if !result.Allowed {
			failedAttempts++
		}

		// After 5 failed attempts, IP should be blocked
		if i >= 5 && result.Allowed {
			t.Errorf("IP should be blocked after %d failed attempts, but request was allowed", i+1)
		}
	}

	if failedAttempts < 10 {
		t.Errorf("Expected at least 10 blocked requests during brute force attack, got %d", failedAttempts)
	}

	// Verify IP is in blocked list
	blockedIPs := securityService.GetBlockedIPs()
	found := false
	for _, blockedIP := range blockedIPs {
		if blockedIP.IP == attackerIP {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Attacker IP %s should be in blocked list", attackerIP)
	}
}

// TestBotSwarmAttack tests bot swarm attack scenario
func TestBotSwarmAttack(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 1000,
			BurstSize:         100,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           true,
			MaxFailedAttempts: 10,
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
	path := "/api/captcha"

	// Different bot user agents
	botUserAgents := []string{
		"bot/1.0",
		"crawler/2.0",
		"spider/3.0",
		"headless-chrome/1.0",
		"python-requests/2.28.0",
		"curl/7.68.0",
		"wget/1.20.3",
		"scrapy/2.5.0",
		"selenium/4.0.0",
		"phantomjs/2.1.1",
	}

	totalRequests := 0
	totalBlocked := 0

	// Launch bot swarm attack
	for i, botUA := range botUserAgents {
		ip := fmt.Sprintf("172.16.%d.%d", i/10, i%10)
		
		for j := 0; j < 50; j++ {
			result, err := securityService.CheckRequest(ctx, ip, botUA, path, time.Millisecond*1, false)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			
			totalRequests++
			if !result.Allowed {
				totalBlocked++
			}
		}
	}

	// Verify that most bot requests are blocked
	blockingPercentage := float64(totalBlocked) / float64(totalRequests) * 100
	t.Logf("Bot swarm attack: %d requests, %d blocked (%.1f%%)", totalRequests, totalBlocked, blockingPercentage)

	if blockingPercentage < 80.0 {
		t.Errorf("Expected at least 80%% of bot requests to be blocked, got %.1f%%", blockingPercentage)
	}
}

// TestSlowLorisAttack tests slow HTTP attack scenario
func TestSlowLorisAttack(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 100,
			BurstSize:         10,
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
	attackerIP := "10.10.10.10"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
	path := "/api/captcha"

	// Simulate slow requests (high response time indicates potential slow attack)
	var wg sync.WaitGroup
	slowRequests := 100
	blockedCount := 0
	var mu sync.Mutex

	for i := 0; i < slowRequests; i++ {
		wg.Add(1)
		go func(requestNum int) {
			defer wg.Done()
			
			// Simulate very slow request processing
			responseTime := time.Second * 5 // 5 seconds - very slow
			result, err := securityService.CheckRequest(ctx, attackerIP, userAgent, path, responseTime, false)
			if err != nil {
				t.Errorf("Unexpected error in slow request %d: %v", requestNum, err)
				return
			}
			
			mu.Lock()
			if !result.Allowed {
				blockedCount++
			}
			mu.Unlock()
		}(i)
		
		// Small delay between requests to simulate slow loris pattern
		time.Sleep(time.Millisecond * 50)
	}

	wg.Wait()

	t.Logf("Slow Loris attack: %d requests, %d blocked", slowRequests, blockedCount)

	// Verify that rate limiting kicks in for slow requests
	if blockedCount < slowRequests/10 {
		t.Errorf("Expected at least %d requests to be blocked during slow attack, got %d", slowRequests/10, blockedCount)
	}
}

// TestMixedAttackScenario tests combined attack scenario
func TestMixedAttackScenario(t *testing.T) {
	securityService := security.NewSecurityService(nil, &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 200,
			BurstSize:         20,
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
	path := "/api/captcha"

	var wg sync.WaitGroup
	var totalRequests, totalBlocked int64
	var mu sync.Mutex

	// Scenario 1: DDoS from multiple IPs
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(ipSuffix int) {
			defer wg.Done()
			ip := fmt.Sprintf("203.0.113.%d", ipSuffix)
			userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
			
			for j := 0; j < 50; j++ {
				result, _ := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*10, false)
				mu.Lock()
				totalRequests++
				if !result.Allowed {
					totalBlocked++
				}
				mu.Unlock()
			}
		}(i)
	}

	// Scenario 2: Bot attacks
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(botIndex int) {
			defer wg.Done()
			ip := fmt.Sprintf("198.51.100.%d", botIndex)
			botUA := fmt.Sprintf("bot-%d/1.0", botIndex)
			
			for j := 0; j < 30; j++ {
				result, _ := securityService.CheckRequest(ctx, ip, botUA, path, time.Millisecond*5, false)
				mu.Lock()
				totalRequests++
				if !result.Allowed {
					totalBlocked++
				}
				mu.Unlock()
			}
		}(i)
	}

	// Scenario 3: Brute force attacks
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(attackerIndex int) {
			defer wg.Done()
			ip := fmt.Sprintf("192.0.2.%d", attackerIndex)
			userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
			
			for j := 0; j < 20; j++ {
				result, _ := securityService.CheckRequest(ctx, ip, userAgent, path, time.Millisecond*20, true) // Failed attempts
				mu.Lock()
				totalRequests++
				if !result.Allowed {
					totalBlocked++
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	blockingPercentage := float64(totalBlocked) / float64(totalRequests) * 100
	t.Logf("Mixed attack scenario: %d total requests, %d blocked (%.1f%%)", totalRequests, totalBlocked, blockingPercentage)

	// In a mixed attack scenario, we expect significant blocking
	if blockingPercentage < 60.0 {
		t.Errorf("Expected at least 60%% of requests to be blocked in mixed attack scenario, got %.1f%%", blockingPercentage)
	}

	// Check that we have blocked IPs
	blockedIPs := securityService.GetBlockedIPs()
	if len(blockedIPs) < 3 {
		t.Errorf("Expected at least 3 IPs to be blocked, got %d", len(blockedIPs))
	}

	// Check security stats
	stats := securityService.GetStats()
	t.Logf("Security stats after mixed attack: %+v", stats)
}
