package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/logger"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/monitoring"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/redis"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/repository"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/transport/grpc"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/usecase"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/websocket"
	pb "github.com/FlooooowY/SteelMount-Captcha-Service/proto/captcha/v1"
	redisLib "github.com/go-redis/redis/v8"
	grpcLib "google.golang.org/grpc"
)

// Server represents the captcha service server
type Server struct {
	config *config.Config
	logger *logrus.Logger

	// gRPC server
	grpcServer *grpcLib.Server
	listener   net.Listener

	// WebSocket server
	wsService *websocket.WebSocketService
	wsServer  *websocket.HTTPServer

	// Security
	redisClient     *redis.Client
	securityService *security.SecurityService
	securityMW      *grpc.SecurityMiddleware

	// Monitoring
	metrics          *monitoring.Metrics
	metricsMW        *monitoring.MetricsMiddleware
	prometheusServer *monitoring.PrometheusServer

	// Balancer integration
	balancerClient *grpc.BalancerClient

	// Server state
	port        int
	wsPort      int
	metricsPort int
	instanceID  string

	// Graceful shutdown
	shutdownWG sync.WaitGroup
	shutdownCh chan struct{}
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	log := logger.GetLogger()

	// Generate instance ID
	instanceID := generateInstanceID()

	srv := &Server{
		config:     cfg,
		logger:     log,
		instanceID: instanceID,
		shutdownCh: make(chan struct{}),
	}

	// Find available ports for gRPC, WebSocket, and metrics in one pass
	ports, err := srv.findAvailablePorts(3)
	if err != nil {
		return nil, fmt.Errorf("failed to find available ports: %w", err)
	}
	srv.port = ports[0]
	srv.wsPort = ports[1] 
	srv.metricsPort = ports[2]

	// Create Redis client with timeout
	redisClient, err := srv.createRedisClientWithTimeout(cfg)
	if err != nil {
		log.Warnf("Failed to create Redis client: %v, using local-only mode", err)
		redisClient = nil
	}
	srv.redisClient = redisClient

	// Create security service
	securityConfig := &security.SecurityConfig{
		RateLimitConfig: security.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: cfg.Security.RateLimit.RequestsPerMinute,
			BurstSize:         cfg.Security.RateLimit.BurstSize,
			Window:            time.Minute,
		},
		IPBlockingConfig: security.IPBlockingConfig{
			Enabled:           cfg.Security.IPBlocking.Enabled,
			MaxFailedAttempts: cfg.Security.IPBlocking.MaxFailedAttempts,
			BlockDuration:     cfg.Security.IPBlocking.BlockDuration,
			CleanupInterval:   cfg.Security.IPBlocking.CleanupInterval,
		},
		BotDetectionConfig: security.BotDetectionConfig{
			Enabled:         cfg.Security.BotDetection.Enabled,
			MinBotScore:     0.4,
			HighBotScore:    0.7,
			CleanupInterval: time.Hour,
		},
	}

	var redisClientForSecurity *redisLib.Client
	if redisClient != nil {
		redisClientForSecurity = redisClient.GetClient()
	}

	srv.securityService = security.NewSecurityService(redisClientForSecurity, securityConfig)

	// Create security middleware
	srv.securityMW = grpc.NewSecurityMiddleware(srv.securityService)

	// Create monitoring with custom registry to avoid duplicate registration
	registry := prometheus.NewRegistry()
	srv.metrics = monitoring.NewMetricsWithRegistry(registry)
	srv.metricsMW = monitoring.NewMetricsMiddleware(srv.metrics)
	srv.prometheusServer = monitoring.NewPrometheusServer(srv.metricsPort, srv.metrics)

	// Create WebSocket service
	srv.wsService = websocket.NewWebSocketService()

	// Create WebSocket HTTP server
	srv.wsServer = websocket.NewHTTPServer(srv.wsService, srv.wsPort)

	// Create gRPC server with security and metrics middleware
	srv.grpcServer = grpcLib.NewServer(
		grpcLib.ChainUnaryInterceptor(
			srv.securityMW.UnaryInterceptor(),
			srv.metricsMW.GRPCMetricsInterceptor(),
		),
		grpcLib.ChainStreamInterceptor(
			srv.securityMW.StreamInterceptor(),
			// Add stream metrics interceptor here if needed
		),
		grpcLib.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpcLib.MaxSendMsgSize(4*1024*1024), // 4MB
	)

	// Create balancer client if enabled (with timeout)
	if cfg.Balancer.Enabled && cfg.Balancer.URL != "" {
		balancerClient, err := srv.createBalancerClientWithTimeout(cfg, instanceID, srv.port)
		if err != nil {
			log.Warnf("Failed to create balancer client: %v, running without balancer", err)
			srv.balancerClient = nil
		} else {
			srv.balancerClient = balancerClient
		}
	}

	// Register services
	srv.registerServices()

	log.Infof("Server created with instance ID: %s, gRPC port: %d, WebSocket port: %d, metrics port: %d", instanceID, srv.port, srv.wsPort, srv.metricsPort)

	return srv, nil
}

// Start starts the server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server...")

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Start gRPC server in a goroutine
	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()

		s.logger.Infof("Starting gRPC server on port %d", s.port)
		if err := s.grpcServer.Serve(listener); err != nil {
			s.logger.Errorf("gRPC server error: %v", err)
		}
	}()

	// Start WebSocket server in a goroutine
	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()

		s.logger.Infof("Starting WebSocket server on port %d", s.wsPort)
		if err := s.wsServer.Start(ctx); err != nil {
			s.logger.Errorf("WebSocket server error: %v", err)
		}
	}()

	// Start Prometheus server in a goroutine
	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()

		s.logger.Infof("Starting Prometheus server on port %d", s.metricsPort)
		if err := s.prometheusServer.Start(ctx); err != nil {
			s.logger.Errorf("Prometheus server error: %v", err)
		}
	}()

	// Start security cleanup routine
	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()
		s.startSecurityCleanup(ctx)
	}()

	// Start balancer registration
	s.shutdownWG.Add(1)
	go func() {
		defer s.shutdownWG.Done()
		s.startBalancerRegistration(ctx)
	}()

	// Wait for context cancellation
	<-ctx.Done()

	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping server...")

	// Signal shutdown
	close(s.shutdownCh)

	// Stop gRPC server gracefully
	grpcDone := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(grpcDone)
	}()

	// Stop WebSocket server gracefully
	wsDone := make(chan struct{})
	go func() {
		if err := s.wsServer.Stop(ctx); err != nil {
			s.logger.Errorf("Error stopping WebSocket server: %v", err)
		}
		close(wsDone)
	}()

	// Stop Prometheus server gracefully
	prometheusDone := make(chan struct{})
	go func() {
		if err := s.prometheusServer.Stop(ctx); err != nil {
			s.logger.Errorf("Error stopping Prometheus server: %v", err)
		}
		close(prometheusDone)
	}()

	// Stop balancer registration
	if s.balancerClient != nil {
		if err := s.balancerClient.StopRegistration(ctx); err != nil {
			s.logger.Errorf("Error stopping balancer registration: %v", err)
		}
		if err := s.balancerClient.Close(); err != nil {
			s.logger.Errorf("Error closing balancer client: %v", err)
		}
	}

	// Close Redis client
	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			s.logger.Errorf("Error closing Redis client: %v", err)
		}
	}

	// Wait for graceful stop or timeout
	select {
	case <-grpcDone:
		s.logger.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Graceful stop timeout, forcing gRPC stop")
		s.grpcServer.Stop()
	}

	select {
	case <-wsDone:
		s.logger.Info("WebSocket server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("WebSocket stop timeout")
	}

	select {
	case <-prometheusDone:
		s.logger.Info("Prometheus server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Prometheus stop timeout")
	}

	// Wait for all goroutines to finish
	waitDone := make(chan struct{})
	go func() {
		s.shutdownWG.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		s.logger.Info("All goroutines stopped")
	case <-ctx.Done():
		s.logger.Warn("Shutdown timeout, some goroutines may still be running")
	}

	return nil
}

// GetPort returns the server port
func (s *Server) GetPort() int {
	return s.port
}

// GetWebSocketPort returns the WebSocket server port
func (s *Server) GetWebSocketPort() int {
	return s.wsPort
}

// GetInstanceID returns the server instance ID
func (s *Server) GetInstanceID() string {
	return s.instanceID
}

// GetWebSocketService returns the WebSocket service
func (s *Server) GetWebSocketService() *websocket.WebSocketService {
	return s.wsService
}

// GetSecurityService returns the security service
func (s *Server) GetSecurityService() *security.SecurityService {
	return s.securityService
}

// GetSecurityStats returns security statistics
func (s *Server) GetSecurityStats() map[string]interface{} {
	stats := s.securityService.GetStats()

	if s.redisClient != nil {
		stats["redis"] = s.redisClient.GetStats()
	}

	return stats
}

// GetMetricsPort returns the metrics server port
func (s *Server) GetMetricsPort() int {
	return s.metricsPort
}

// GetMetrics returns the metrics instance
func (s *Server) GetMetrics() *monitoring.Metrics {
	return s.metrics
}

// GetMetricsMiddleware returns the metrics middleware
func (s *Server) GetMetricsMiddleware() *monitoring.MetricsMiddleware {
	return s.metricsMW
}

// findAvailablePort finds an available port in the configured range
func (s *Server) findAvailablePort() (int, error) {
	return s.findAvailablePortFrom(s.config.Server.MinPort)
}

// findAvailablePortFrom finds an available port starting from the specified port
func (s *Server) findAvailablePortFrom(startPort int) (int, error) {
	for port := startPort; port <= s.config.Server.MaxPort; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			// Add delay to ensure port is fully released
			time.Sleep(50 * time.Millisecond)
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", startPort, s.config.Server.MaxPort)
}

// registerServices registers gRPC services
func (s *Server) registerServices() {
	// Register gRPC services
	challengeRepo := repository.NewInMemoryChallengeRepository()
	usecaseConfig := &usecase.Config{
		MaxActiveChallenges: s.config.Captcha.MaxActiveChallenges,
		ChallengeTimeout:    time.Minute * 5,  // 5 minutes timeout
		CleanupInterval:     time.Minute * 10, // Cleanup every 10 minutes
	}
	captchaUsecase := usecase.NewCaptchaUsecase(challengeRepo, usecaseConfig)
	captchaService := grpc.NewCaptchaService(captchaUsecase)

	pb.RegisterCaptchaServiceServer(s.grpcServer, captchaService)
	s.logger.Info("Services registered")
}

// startBalancerRegistration starts the balancer registration process
func (s *Server) startBalancerRegistration(ctx context.Context) {
	if s.balancerClient == nil {
		s.logger.Info("Balancer client not configured, skipping registration")
		return
	}

	s.logger.Info("Starting balancer registration")
	
	// Start registration with retries
	for attempt := 1; attempt <= s.config.Balancer.MaxRetryAttempts; attempt++ {
		if err := s.balancerClient.StartRegistration(ctx); err != nil {
			s.logger.Errorf("Failed to register with balancer (attempt %d/%d): %v", 
				attempt, s.config.Balancer.MaxRetryAttempts, err)
			
			if attempt < s.config.Balancer.MaxRetryAttempts {
				select {
				case <-ctx.Done():
					return
				case <-time.After(s.config.Balancer.RetryDelay):
					continue
				}
			}
		} else {
			s.logger.Info("Successfully registered with balancer")
			break
		}
	}
}

// startSecurityCleanup starts the security cleanup routine
func (s *Server) startSecurityCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.config.Security.RateLimit.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Security cleanup stopped")
			return
		case <-s.shutdownCh:
			s.logger.Info("Security cleanup stopped due to shutdown")
			return
		case <-ticker.C:
			s.logger.Debug("Running security cleanup")
			s.securityService.Cleanup()
		}
	}
}

// generateInstanceID generates a unique instance ID
func generateInstanceID() string {
	return fmt.Sprintf("captcha-%d", time.Now().UnixNano())
}

// createRedisClientWithTimeout creates Redis client with connection timeout
func (s *Server) createRedisClientWithTimeout(cfg *config.Config) (*redis.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.InitTimeout)
	defer cancel()

	// Create a channel to receive the result
	resultCh := make(chan struct {
		client *redis.Client
		err    error
	}, 1)

	go func() {
		client, err := redis.NewClient(&cfg.Redis)
		resultCh <- struct {
			client *redis.Client
			err    error
		}{client, err}
	}()

	select {
	case result := <-resultCh:
		return result.client, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("redis client creation timeout after %v", cfg.Server.InitTimeout)
	}
}

// createBalancerClientWithTimeout creates balancer client with connection timeout
func (s *Server) createBalancerClientWithTimeout(cfg *config.Config, instanceID string, port int) (*grpc.BalancerClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.InitTimeout)
	defer cancel()

	// Create a channel to receive the result
	resultCh := make(chan struct {
		client *grpc.BalancerClient
		err    error
	}, 1)

	go func() {
		client, err := grpc.NewBalancerClient(
			cfg.Balancer.URL,
			instanceID,
			"localhost", // host
			"interactive", // challenge type
			port,
		)
		resultCh <- struct {
			client *grpc.BalancerClient
			err    error
		}{client, err}
	}()

	select {
	case result := <-resultCh:
		return result.client, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("balancer client creation timeout after %v", cfg.Server.InitTimeout)
	}
}

// findAvailablePorts finds multiple consecutive available ports efficiently
func (s *Server) findAvailablePorts(count int) ([]int, error) {
	minPort := s.config.Server.MinPort
	maxPort := s.config.Server.MaxPort
	
	for port := minPort; port <= maxPort-count+1; port++ {
		// Check if we can get 'count' consecutive ports starting from 'port'
		availablePorts := make([]int, 0, count)
		allAvailable := true
		
		for i := 0; i < count; i++ {
			currentPort := port + i
			if currentPort > maxPort {
				allAvailable = false
				break
			}
			
			// Quick check if port is available
			if !s.isPortAvailable(currentPort) {
				allAvailable = false
				break
			}
			availablePorts = append(availablePorts, currentPort)
		}
		
		if allAvailable && len(availablePorts) == count {
			return availablePorts, nil
		}
	}
	
	return nil, fmt.Errorf("could not find %d consecutive available ports in range %d-%d", count, minPort, maxPort)
}

// isPortAvailable checks if a port is available
func (s *Server) isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
