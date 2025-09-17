package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Request metrics
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestsInFlight prometheus.Gauge

	// Captcha metrics
	CaptchaGenerated  *prometheus.CounterVec
	CaptchaValidated  *prometheus.CounterVec
	CaptchaErrors     *prometheus.CounterVec
	ActiveChallenges  prometheus.Gauge
	ChallengeDuration *prometheus.HistogramVec

	// Security metrics
	SecurityBlocks *prometheus.CounterVec
	RateLimitHits  *prometheus.CounterVec
	BotDetections  *prometheus.CounterVec
	BlockedIPs     prometheus.Gauge

	// Performance metrics
	MemoryUsage  prometheus.Gauge
	CPUUsage     prometheus.Gauge
	RPS          prometheus.Gauge
	ResponseTime *prometheus.HistogramVec

	// WebSocket metrics
	WebSocketConnections prometheus.Gauge
	WebSocketEvents      *prometheus.CounterVec
	WebSocketErrors      *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return newMetricsWithRegistry(prometheus.DefaultRegisterer)
}

// NewMetricsWithRegistry creates a new metrics instance with custom registry
func NewMetricsWithRegistry(registry prometheus.Registerer) *Metrics {
	return newMetricsWithRegistry(registry)
}

// newMetricsWithRegistry creates metrics with specified registry
func newMetricsWithRegistry(registry prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		// Request metrics
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_requests_total",
				Help: "Total number of requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "captcha_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		RequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_requests_in_flight",
				Help: "Number of requests currently being processed",
			},
		),

		// Captcha metrics
		CaptchaGenerated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_generated_total",
				Help: "Total number of captchas generated",
			},
			[]string{"type", "complexity"},
		),
		CaptchaValidated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_validated_total",
				Help: "Total number of captchas validated",
			},
			[]string{"type", "result"},
		),
		CaptchaErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_errors_total",
				Help: "Total number of captcha errors",
			},
			[]string{"type", "error"},
		),
		ActiveChallenges: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_active_challenges",
				Help: "Number of active captcha challenges",
			},
		),
		ChallengeDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "captcha_challenge_duration_seconds",
				Help:    "Captcha challenge duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
			},
			[]string{"type"},
		),

		// Security metrics
		SecurityBlocks: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_security_blocks_total",
				Help: "Total number of security blocks",
			},
			[]string{"reason", "type"},
		),
		RateLimitHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_rate_limit_hits_total",
				Help: "Total number of rate limit hits",
			},
			[]string{"ip", "endpoint"},
		),
		BotDetections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_bot_detections_total",
				Help: "Total number of bot detections",
			},
			[]string{"ip", "reason"},
		),
		BlockedIPs: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_blocked_ips",
				Help: "Number of currently blocked IPs",
			},
		),

		// Performance metrics
		MemoryUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_memory_usage_bytes",
				Help: "Current memory usage in bytes",
			},
		),
		CPUUsage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_cpu_usage_percent",
				Help: "Current CPU usage percentage",
			},
		),
		RPS: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_requests_per_second",
				Help: "Current requests per second",
			},
		),
		ResponseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "captcha_response_time_seconds",
				Help:    "Response time in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"endpoint"},
		),

		// WebSocket metrics
		WebSocketConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "captcha_websocket_connections",
				Help: "Number of active WebSocket connections",
			},
		),
		WebSocketEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_websocket_events_total",
				Help: "Total number of WebSocket events",
			},
			[]string{"type", "client_id"},
		),
		WebSocketErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_websocket_errors_total",
				Help: "Total number of WebSocket errors",
			},
			[]string{"type", "error"},
		),
	}

	// Register all metrics with the registry
	registry.MustRegister(
		metrics.RequestsTotal,
		metrics.RequestDuration,
		metrics.RequestsInFlight,
		metrics.CaptchaGenerated,
		metrics.CaptchaValidated,
		metrics.CaptchaErrors,
		metrics.ActiveChallenges,
		metrics.ChallengeDuration,
		metrics.SecurityBlocks,
		metrics.RateLimitHits,
		metrics.BotDetections,
		metrics.BlockedIPs,
		metrics.MemoryUsage,
		metrics.CPUUsage,
		metrics.RPS,
		metrics.ResponseTime,
		metrics.WebSocketConnections,
		metrics.WebSocketEvents,
		metrics.WebSocketErrors,
	)

	return metrics
}

// RecordRequest records a request metric
func (m *Metrics) RecordRequest(method, endpoint, status string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.RequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordCaptchaGenerated records a captcha generation
func (m *Metrics) RecordCaptchaGenerated(captchaType, complexity string) {
	m.CaptchaGenerated.WithLabelValues(captchaType, complexity).Inc()
}

// RecordCaptchaValidated records a captcha validation
func (m *Metrics) RecordCaptchaValidated(captchaType, result string) {
	m.CaptchaValidated.WithLabelValues(captchaType, result).Inc()
}

// RecordCaptchaError records a captcha error
func (m *Metrics) RecordCaptchaError(captchaType, errorType string) {
	m.CaptchaErrors.WithLabelValues(captchaType, errorType).Inc()
}

// SetActiveChallenges sets the number of active challenges
func (m *Metrics) SetActiveChallenges(count int) {
	m.ActiveChallenges.Set(float64(count))
}

// RecordChallengeDuration records challenge duration
func (m *Metrics) RecordChallengeDuration(captchaType string, duration time.Duration) {
	m.ChallengeDuration.WithLabelValues(captchaType).Observe(duration.Seconds())
}

// RecordSecurityBlock records a security block
func (m *Metrics) RecordSecurityBlock(reason, blockType string) {
	m.SecurityBlocks.WithLabelValues(reason, blockType).Inc()
}

// RecordRateLimitHit records a rate limit hit
func (m *Metrics) RecordRateLimitHit(ip, endpoint string) {
	m.RateLimitHits.WithLabelValues(ip, endpoint).Inc()
}

// RecordBotDetection records a bot detection
func (m *Metrics) RecordBotDetection(ip, reason string) {
	m.BotDetections.WithLabelValues(ip, reason).Inc()
}

// SetBlockedIPs sets the number of blocked IPs
func (m *Metrics) SetBlockedIPs(count int) {
	m.BlockedIPs.Set(float64(count))
}

// SetMemoryUsage sets memory usage
func (m *Metrics) SetMemoryUsage(bytes int64) {
	m.MemoryUsage.Set(float64(bytes))
}

// SetCPUUsage sets CPU usage
func (m *Metrics) SetCPUUsage(percent float64) {
	m.CPUUsage.Set(percent)
}

// SetRPS sets requests per second
func (m *Metrics) SetRPS(rps float64) {
	m.RPS.Set(rps)
}

// RecordResponseTime records response time
func (m *Metrics) RecordResponseTime(endpoint string, duration time.Duration) {
	m.ResponseTime.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// SetWebSocketConnections sets WebSocket connections count
func (m *Metrics) SetWebSocketConnections(count int) {
	m.WebSocketConnections.Set(float64(count))
}

// RecordWebSocketEvent records a WebSocket event
func (m *Metrics) RecordWebSocketEvent(eventType, clientID string) {
	m.WebSocketEvents.WithLabelValues(eventType, clientID).Inc()
}

// RecordWebSocketError records a WebSocket error
func (m *Metrics) RecordWebSocketError(eventType, errorType string) {
	m.WebSocketErrors.WithLabelValues(eventType, errorType).Inc()
}
