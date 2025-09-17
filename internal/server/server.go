package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/logger"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/monitoring"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/redis"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/security"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/transport/grpc"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/websocket"
	"google.golang.org/grpc"
)

// Server represents the captcha service server
type Server struct {
	config *config.Config
	logger *logger.Logger

	// gRPC server
	grpcServer *grpc.Server
	listener   net.Listener

	// WebSocket server
	wsService *websocket.WebSocketService
	wsServer  *websocket.HTTPServer

	// Security
	redisClient     *redis.Client
	securityService *security.SecurityService
	securityMW      *grpc.SecurityMiddleware

	// Monitoring
	metrics         *monitoring.Metrics
	metricsMW       *monitoring.MetricsMiddleware
	prometheusServer *monitoring.PrometheusServer

	// Server state
	port       int
	wsPort     int
	metricsPort int
	instanceID string

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

	// Find available port for gRPC
	port, err := srv.findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	srv.port = port

	// Find available port for WebSocket
	wsPort, err := srv.findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available WebSocket port: %w", err)
	}
	srv.wsPort = wsPort

	// Find available port for metrics
	metricsPort, err := srv.findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available metrics port: %w", err)
	}
	srv.metricsPort = metricsPort

	// Create Redis client
	redisClient, err := redis.NewClient(&cfg.Redis)
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
	
	var redisClientForSecurity *redis.Client
	if redisClient != nil {
		redisClientForSecurity = redisClient.GetClient()
	}
	
	srv.securityService = security.NewSecurityService(redisClientForSecurity, securityConfig)

	// Create security middleware
	srv.securityMW = grpc.NewSecurityMiddleware(srv.securityService)

	// Create monitoring
	srv.metrics = monitoring.NewMetrics()
	srv.metricsMW = monitoring.NewMetricsMiddleware(srv.metrics)
	srv.prometheusServer = monitoring.NewPrometheusServer(srv.metricsPort, srv.metrics)

	// Create WebSocket service
	srv.wsService = websocket.NewWebSocketService()

	// Create WebSocket HTTP server
	srv.wsServer = websocket.NewHTTPServer(srv.wsService, srv.wsPort)

	// Create gRPC server with security and metrics middleware
	srv.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			srv.securityMW.UnaryInterceptor(),
			srv.metricsMW.GRPCMetricsInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			srv.securityMW.StreamInterceptor(),
			// Add stream metrics interceptor here if needed
		),
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB
	)

	// Register services
	srv.registerServices()

	log.Infof("Server created with instance ID: %s, gRPC port: %d, WebSocket port: %d, metrics port: %d", instanceID, port, wsPort, srv.metricsPort)

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
	for port := s.config.Server.MinPort; port <= s.config.Server.MaxPort; port++ {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", s.config.Server.MinPort, s.config.Server.MaxPort)
}

// registerServices registers gRPC services
func (s *Server) registerServices() {
	// TODO: Register captcha service when usecase is ready
	s.logger.Info("Services registered")
}

// startBalancerRegistration starts the balancer registration process
func (s *Server) startBalancerRegistration(ctx context.Context) {
	ticker := time.NewTicker(s.config.Balancer.RegistrationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Balancer registration stopped")
			return
		case <-s.shutdownCh:
			s.logger.Info("Balancer registration stopped due to shutdown")
			return
		case <-ticker.C:
			s.logger.Debug("Sending heartbeat to balancer")
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
