package integration

import (
	"context"
	"testing"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/server"
)

func TestServer_StartStop(t *testing.T) {
	// Create test config
	cfg := &config.Config{
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
				Enabled:             true,
				MaxFailedAttempts:   5,
				BlockDuration:       5 * time.Minute,
				CleanupInterval:     5 * time.Minute,
			},
			BotDetection: config.BotDetectionConfig{
				Enabled:         true,
				SuspiciousPatterns: []string{"bot", "crawler", "spider"},
			},
		},
		Monitoring: config.MonitoringConfig{
			PrometheusPort: 9090,
			MetricsPath:    "/metrics",
			HealthCheckPath: "/health",
		},
		Balancer: config.BalancerConfig{
			RegistrationInterval: 1 * time.Second,
			HeartbeatTimeout:     30 * time.Second,
			MaxRetryAttempts:     3,
			RetryDelay:           5 * time.Second,
		},
	}
	
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Start server in goroutine
	startErr := make(chan error, 1)
	go func() {
		startErr <- srv.Start(ctx)
	}()
	
	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)
	
	// Test server stop
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	
	err = srv.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
	
	// Check if start returned an error (context cancelled is expected)
	select {
	case err := <-startErr:
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected start error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Start goroutine didn't return")
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
	// Create multiple servers to test port discovery
	cfg := &config.Config{
		Server: config.ServerConfig{
			MinPort: 38000,
			MaxPort: 38010, // Small range for testing
		},
		Redis: config.RedisConfig{
			URL: "redis://localhost:6379",
		},
		Captcha: config.CaptchaConfig{
			MaxActiveChallenges: 100,
			MemoryLimitGB:       1,
			TargetRPS:           10,
			ChallengeTimeout:    1 * time.Minute,
			CleanupInterval:     1 * time.Minute,
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				RequestsPerMinute: 60,
				BurstSize:         10,
				CleanupInterval:   1 * time.Minute,
			},
			IPBlocking: config.IPBlockingConfig{
				Enabled:           false, // Disable for testing
				MaxFailedAttempts: 5,
				BlockDuration:     5 * time.Minute,
				CleanupInterval:   5 * time.Minute,
			},
			BotDetection: config.BotDetectionConfig{
				Enabled: false, // Disable for testing
			},
		},
		Monitoring: config.MonitoringConfig{
			PrometheusPort: 9090,
			MetricsPath:    "/metrics",
		},
		Balancer: config.BalancerConfig{
			RegistrationInterval: 1 * time.Second,
			HeartbeatTimeout:     30 * time.Second,
			MaxRetryAttempts:     3,
			RetryDelay:           5 * time.Second,
		},
	}
	
	// Create first server
	srv1, err := server.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create first server: %v", err)
	}
	
	// Create second server
	srv2, err := server.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create second server: %v", err)
	}
	
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
