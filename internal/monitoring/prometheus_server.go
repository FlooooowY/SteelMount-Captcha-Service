package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusServer provides Prometheus metrics server
type PrometheusServer struct {
	server  *http.Server
	port    int
	metrics *Metrics
}

// NewPrometheusServer creates a new Prometheus server
func NewPrometheusServer(port int, metrics *Metrics) *PrometheusServer {
	return &PrometheusServer{
		port:    port,
		metrics: metrics,
	}
}

// Start starts the Prometheus server
func (ps *PrometheusServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	mux.HandleFunc("/health", ps.healthHandler)

	// Custom metrics endpoint
	mux.HandleFunc("/custom-metrics", ps.customMetricsHandler)

	ps.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ps.port),
		Handler: mux,
	}

	// Start metrics collection
	go ps.collectSystemMetrics(ctx)

	// Start server in goroutine
	go func() {
		if err := ps.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus server error: %v\n", err)
		}
	}()

	fmt.Printf("Prometheus server started on port %d\n", ps.port)
	return nil
}

// Stop stops the Prometheus server
func (ps *PrometheusServer) Stop(ctx context.Context) error {
	if ps.server != nil {
		return ps.server.Shutdown(ctx)
	}
	return nil
}

// healthHandler handles health check requests
func (ps *PrometheusServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Unix(),
		"port":   ps.port,
	}

	fmt.Fprintf(w, `{"status":"%s","time":%d,"port":%d}`,
		response["status"], response["time"], response["port"])
}

// customMetricsHandler handles custom metrics requests
func (ps *PrometheusServer) customMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Get system metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := map[string]interface{}{
		"memory_usage_bytes": m.Alloc,
		"memory_sys_bytes":   m.Sys,
		"gc_runs":            m.NumGC,
		"goroutines":         runtime.NumGoroutine(),
		"timestamp":          time.Now().Unix(),
	}

	fmt.Fprintf(w, `{"memory_usage_bytes":%d,"memory_sys_bytes":%d,"gc_runs":%d,"goroutines":%d,"timestamp":%d}`,
		metrics["memory_usage_bytes"], metrics["memory_sys_bytes"],
		metrics["gc_runs"], metrics["goroutines"], metrics["timestamp"])
}

// collectSystemMetrics collects system metrics periodically
func (ps *PrometheusServer) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ps.updateSystemMetrics()
		}
	}
}

// updateSystemMetrics updates system metrics
func (ps *PrometheusServer) updateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Update memory usage
	ps.metrics.SetMemoryUsage(int64(m.Alloc))

	// Update CPU usage (simplified)
	cpuUsage := float64(runtime.NumGoroutine()) * 0.1 // Rough estimate
	ps.metrics.SetCPUUsage(cpuUsage)
}
