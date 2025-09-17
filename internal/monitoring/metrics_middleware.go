package monitoring

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

// MetricsMiddleware provides middleware for collecting metrics
type MetricsMiddleware struct {
	metrics *Metrics
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(metrics *Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
	}
}

// HTTPMiddleware creates HTTP middleware for metrics collection
func (mm *MetricsMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Increment requests in flight
		mm.metrics.RequestsInFlight.Inc()
		defer mm.metrics.RequestsInFlight.Dec()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start)
		status := http.StatusText(wrapped.statusCode)

		mm.metrics.RecordRequest(r.Method, r.URL.Path, status, duration)
		mm.metrics.RecordResponseTime(r.URL.Path, duration)
	})
}

// GRPCMetricsInterceptor creates gRPC interceptor for metrics collection
func (mm *MetricsMiddleware) GRPCMetricsInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Increment requests in flight
		mm.metrics.RequestsInFlight.Inc()
		defer mm.metrics.RequestsInFlight.Dec()

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start)
		status := "success"
		if err != nil {
			status = "error"
		}

		mm.metrics.RecordRequest("grpc", info.FullMethod, status, duration)
		mm.metrics.RecordResponseTime(info.FullMethod, duration)

		return resp, err
	}
}

// WebSocketMetricsInterceptor creates WebSocket interceptor for metrics collection
func (mm *MetricsMiddleware) WebSocketMetricsInterceptor() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Increment WebSocket connections
			mm.metrics.WebSocketConnections.Inc()
			defer mm.metrics.WebSocketConnections.Dec()

			// Call next handler
			next.ServeHTTP(w, r)

			// Record WebSocket connection duration
			duration := time.Since(start)
			mm.metrics.RecordResponseTime("websocket", duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RecordCaptchaMetrics records captcha-related metrics
func (mm *MetricsMiddleware) RecordCaptchaMetrics(captchaType string, action string, duration time.Duration, success bool) {
	switch action {
	case "generate":
		mm.metrics.RecordCaptchaGenerated(captchaType, "normal")
		mm.metrics.RecordChallengeDuration(captchaType, duration)
	case "validate":
		result := "success"
		if !success {
			result = "failure"
		}
		mm.metrics.RecordCaptchaValidated(captchaType, result)
	case "error":
		mm.metrics.RecordCaptchaError(captchaType, "validation")
	}
}

// RecordSecurityMetrics records security-related metrics
func (mm *MetricsMiddleware) RecordSecurityMetrics(blockType, reason, ip, endpoint string) {
	switch blockType {
	case "block":
		mm.metrics.RecordSecurityBlock(reason, "ip_block")
	case "rate_limit":
		mm.metrics.RecordRateLimitHit(ip, endpoint)
	case "bot_detection":
		mm.metrics.RecordBotDetection(ip, reason)
	}
}

// UpdateActiveChallenges updates the active challenges count
func (mm *MetricsMiddleware) UpdateActiveChallenges(count int) {
	mm.metrics.SetActiveChallenges(count)
}

// UpdateBlockedIPs updates the blocked IPs count
func (mm *MetricsMiddleware) UpdateBlockedIPs(count int) {
	mm.metrics.SetBlockedIPs(count)
}

// UpdateRPS updates the requests per second
func (mm *MetricsMiddleware) UpdateRPS(rps float64) {
	mm.metrics.SetRPS(rps)
}

// RecordWebSocketEvent records a WebSocket event
func (mm *MetricsMiddleware) RecordWebSocketEvent(eventType, clientID string) {
	mm.metrics.RecordWebSocketEvent(eventType, clientID)
}

// RecordWebSocketError records a WebSocket error
func (mm *MetricsMiddleware) RecordWebSocketError(eventType, errorType string) {
	mm.metrics.RecordWebSocketError(eventType, errorType)
}
