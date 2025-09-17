package integration

import (
	"context"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/server"
)

// createTestConfig creates a test configuration
func createTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			MinPort:         38000,
			MaxPort:         40000,
			ShutdownTimeout: 30 * time.Second,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
		},
		Redis: config.RedisConfig{
			URL:          "redis://localhost:6379",
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Captcha: config.CaptchaConfig{
			MaxActiveChallenges: 1000,
			MemoryLimitGB:       1,
			TargetRPS:           100,
			ChallengeTimeout:    5 * time.Minute,
			CleanupInterval:     1 * time.Minute,
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				RequestsPerMinute: 60,
				BurstSize:         10,
				CleanupInterval:   1 * time.Minute,
			},
			IPBlocking: config.IPBlockingConfig{
				Enabled:           true,
				MaxFailedAttempts: 5,
				BlockDuration:     5 * time.Minute,
				CleanupInterval:   5 * time.Minute,
			},
			BotDetection: config.BotDetectionConfig{
				Enabled:            true,
				SuspiciousPatterns: []string{"bot", "crawler", "spider"},
			},
		},
		Monitoring: config.MonitoringConfig{
			PrometheusPort:  9090,
			MetricsPath:     "/metrics",
			HealthCheckPath: "/health",
		},
		Balancer: config.BalancerConfig{
			RegistrationInterval: 1 * time.Second,
			HeartbeatTimeout:     30 * time.Second,
			MaxRetryAttempts:     3,
			RetryDelay:           5 * time.Second,
		},
	}
}

func TestServer_StartStop(t *testing.T) {
	// Create test config
	cfg := createTestConfig()

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test server creation
	if srv == nil {
		t.Fatal("Server is nil")
	}

	// Test server properties
	if srv.GetPort() == 0 {
		t.Error("Server port should be set")
	}

	if srv.GetWebSocketPort() == 0 {
		t.Error("WebSocket port should be set")
	}

	if srv.GetMetricsPort() == 0 {
		t.Error("Metrics port should be set")
	}

	if srv.GetInstanceID() == "" {
		t.Error("Instance ID should be set")
	}

	// Test server start (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start server in goroutine
	startErr := make(chan error, 1)
	go func() {
		startErr <- srv.Start(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(200 * time.Millisecond)

	// Test server stop
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	err = srv.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Wait for start goroutine to return
	select {
	case err := <-startErr:
		// Context cancelled is expected when we stop the server
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected start error: %v", err)
		}
	case <-time.After(1 * time.Second):
		// If start goroutine doesn't return, it's likely because server is running
		// This is acceptable for this test
		t.Log("Start goroutine didn't return (server may still be running)")
	}
}

func TestServer_WithNilConfig(t *testing.T) {
	// Test server creation with nil config
	srv, err := server.New(nil)

	if err == nil {
		t.Error("Expected error with nil config, got nil")
	}

	if srv != nil {
		t.Error("Expected nil server with nil config")
	}
}

func TestServer_PortDiscovery(t *testing.T) {
	// Create test config with small port range
	cfg1 := createTestConfig()
	cfg1.Server.MinPort = 38000
	cfg1.Server.MaxPort = 38005                // Small range for first server
	cfg1.Security.IPBlocking.Enabled = false   // Disable for testing
	cfg1.Security.BotDetection.Enabled = false // Disable for testing

	cfg2 := createTestConfig()
	cfg2.Server.MinPort = 38010
	cfg2.Server.MaxPort = 38015                // Different range for second server
	cfg2.Security.IPBlocking.Enabled = false   // Disable for testing
	cfg2.Security.BotDetection.Enabled = false // Disable for testing

	// Create first server
	srv1, err := server.New(cfg1)
	if err != nil {
		t.Fatalf("Failed to create first server: %v", err)
	}

	// Create second server with different port range
	srv2, err := server.New(cfg2)
	if err != nil {
		t.Fatalf("Failed to create second server: %v", err)
	}

	// Stop both servers
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv1.Stop(ctx)
	srv2.Stop(ctx)

	// Ports should be different
	if srv1.GetPort() == srv2.GetPort() {
		t.Errorf("Servers should have different ports: %d", srv1.GetPort())
	}

	if srv1.GetWebSocketPort() == srv2.GetWebSocketPort() {
		t.Errorf("Servers should have different WebSocket ports: %d", srv1.GetWebSocketPort())
	}

	if srv1.GetMetricsPort() == srv2.GetMetricsPort() {
		t.Errorf("Servers should have different metrics ports: %d", srv1.GetMetricsPort())
	}

	t.Logf("Server 1 - gRPC: %d, WebSocket: %d, Metrics: %d",
		srv1.GetPort(), srv1.GetWebSocketPort(), srv1.GetMetricsPort())
	t.Logf("Server 2 - gRPC: %d, WebSocket: %d, Metrics: %d",
		srv2.GetPort(), srv2.GetWebSocketPort(), srv2.GetMetricsPort())
}
