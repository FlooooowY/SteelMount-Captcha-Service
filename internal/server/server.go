package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/logger"
	"google.golang.org/grpc"
)

// Server represents the captcha service server
type Server struct {
	config *config.Config
	logger *logger.Logger

	// gRPC server
	grpcServer *grpc.Server
	listener   net.Listener

	// Server state
	port       int
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

	// Find available port
	port, err := srv.findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}
	srv.port = port

	// Create gRPC server
	srv.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB
	)

	// Register services
	srv.registerServices()

	log.Infof("Server created with instance ID: %s, port: %d", instanceID, port)

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
	done := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(done)
	}()

	// Wait for graceful stop or timeout
	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Graceful stop timeout, forcing stop")
		s.grpcServer.Stop()
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

// GetInstanceID returns the server instance ID
func (s *Server) GetInstanceID() string {
	return s.instanceID
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

// generateInstanceID generates a unique instance ID
func generateInstanceID() string {
	return fmt.Sprintf("captcha-%d", time.Now().UnixNano())
}
