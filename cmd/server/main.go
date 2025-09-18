package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/logger"
	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(
		cfg.Monitoring.Logging.Level,
		cfg.Monitoring.Logging.Format,
		cfg.Monitoring.Logging.Output,
	)

	log := logger.GetLogger()
	log.Info("Starting SteelMount Captcha Service")

	// Create server instance
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in a goroutine with startup timeout
	startupCtx, startupCancel := context.WithTimeout(ctx, cfg.Server.StartupTimeout)
	defer startupCancel()
	
	go func() {
		if err := srv.Start(startupCtx); err != nil {
			log.Errorf("Server error: %v", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Infof("Received signal %v, shutting down gracefully...", sig)
		cancel()
	case <-ctx.Done():
		log.Info("Server context cancelled")
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Errorf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	log.Info("Server stopped gracefully")
}
