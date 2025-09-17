package security

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
)

// OptimizedBotDetector provides high-performance bot detection with caching
type OptimizedBotDetector struct {
	mu                  sync.RWMutex
	suspiciousPatterns  []*regexp.Regexp
	behaviorPatterns    map[string]*OptimizedBehaviorPattern
	userAgents         map[string]*UserAgentStats
	requestPatterns    map[string]*OptimizedRequestPattern
	
	// Caching for performance
	patternCache       map[string]bool
	cacheMu            sync.RWMutex
	cacheExpiry        time.Time
	cacheDuration      time.Duration
}

// OptimizedBehaviorPattern represents an optimized behavior pattern
type OptimizedBehaviorPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Weight      float64
	Description string
	// Pre-compiled for performance
	CompiledPattern *regexp.Regexp
}

// OptimizedRequestPattern tracks request patterns with optimizations
type OptimizedRequestPattern struct {
	IP              string
	RequestCount    int
	FirstRequest    time.Time
	LastRequest     time.Time
	UserAgents      map[string]int
	RequestPaths    map[string]int
	ResponseTimes   []time.Duration
	ErrorCount      int
	// Pre-allocated slices for performance
	responseTimeBuffer []time.Duration
	mu                sync.RWMutex
}

// UserAgentStats tracks user agent statistics
type UserAgentStats struct {
	Count     int
	LastSeen  time.Time
	BotScore  float64
	IsBot     bool
}

// NewOptimizedBotDetector creates a new optimized bot detector
func NewOptimizedBotDetector() *OptimizedBotDetector {
	bd := &OptimizedBotDetector{
		behaviorPatterns: make(map[string]*OptimizedBehaviorPattern),
		userAgents:      make(map[string]*UserAgentStats),
		requestPatterns: make(map[string]*OptimizedRequestPattern),
		patternCache:    make(map[string]bool),
		cacheDuration:   5 * time.Minute,
		cacheExpiry:     time.Now().Add(5 * time.Minute),
	}
	
	// Initialize suspicious patterns with pre-compiled regexes
	bd.initializeOptimizedPatterns()
	
	return bd
}

// AnalyzeRequest analyzes a request for bot behavior with optimizations
func (obd *OptimizedBotDetector) AnalyzeRequest(ctx context.Context, ip string, userAgent string, path string, responseTime time.Duration, isError bool) (*BotScore, error) {
	obd.mu.Lock()
	defer obd.mu.Unlock()
	
	// Get or create request pattern for IP
	pattern, exists := obd.requestPatterns[ip]
	if !exists {
		pattern = &OptimizedRequestPattern{
			IP:                 ip,
			UserAgents:         make(map[string]int),
			RequestPaths:       make(map[string]int),
			ResponseTimes:      make([]time.Duration, 0, 100), // Pre-allocate
			responseTimeBuffer: make([]time.Duration, 0, 100),
		}
		obd.requestPatterns[ip] = pattern
	}
	
	// Update pattern with optimizations
	obd.updateOptimizedPattern(pattern, userAgent, path, responseTime, isError)
	
	// Calculate bot score with caching
	score := obd.calculateOptimizedBotScore(ip, userAgent, path, pattern)
	
	return score, nil
}

// updateOptimizedPattern updates request pattern with optimizations
func (obd *OptimizedBotDetector) updateOptimizedPattern(pattern *OptimizedRequestPattern, userAgent string, path string, responseTime time.Duration, isError bool) {
	pattern.mu.Lock()
	defer pattern.mu.Unlock()
	
	now := time.Now()
	if pattern.FirstRequest.IsZero() {
		pattern.FirstRequest = now
	}
	pattern.LastRequest = now
	pattern.RequestCount++
	pattern.UserAgents[userAgent]++
	pattern.RequestPaths[path]++
	
	// Use pre-allocated buffer for response times
	if len(pattern.ResponseTimes) < cap(pattern.ResponseTimes) {
		pattern.ResponseTimes = append(pattern.ResponseTimes, responseTime)
	} else {
		// Shift array to make room (circular buffer)
		copy(pattern.ResponseTimes, pattern.ResponseTimes[1:])
		pattern.ResponseTimes[len(pattern.ResponseTimes)-1] = responseTime
	}
	
	if isError {
		pattern.ErrorCount++
	}
}

// calculateOptimizedBotScore calculates bot score with optimizations
func (obd *OptimizedBotDetector) calculateOptimizedBotScore(ip string, userAgent string, path string, pattern *OptimizedRequestPattern) *BotScore {
	score := 0.0
	reasons := make([]string, 0, 10) // Pre-allocate with capacity
	
	// Check user agent patterns with caching
	uaScore, uaReasons := obd.analyzeOptimizedUserAgent(userAgent)
	score += uaScore
	reasons = append(reasons, uaReasons...)
	
	// Check request frequency with optimizations
	freqScore, freqReasons := obd.analyzeOptimizedRequestFrequency(pattern)
	score += freqScore
	reasons = append(reasons, freqReasons...)
	
	// Check response time patterns
	timeScore, timeReasons := obd.analyzeOptimizedResponseTimes(pattern)
	score += timeScore
	reasons = append(reasons, timeReasons...)
	
	// Check error patterns
	errorScore, errorReasons := obd.analyzeOptimizedErrorPatterns(pattern)
	score += errorScore
	reasons = append(reasons, errorReasons...)
	
	// Check path patterns
	pathScore, pathReasons := obd.analyzeOptimizedPathPatterns(path, pattern)
	score += pathScore
	reasons = append(reasons, pathReasons...)
	
	// Calculate confidence based on data points
	confidence := obd.calculateOptimizedConfidence(pattern)
	
	return &BotScore{
		IP:         ip,
		Score:      score,
		Confidence: confidence,
		Reasons:    reasons,
		Timestamp:  time.Now(),
	}
}

// analyzeOptimizedUserAgent analyzes user agent with caching
func (obd *OptimizedBotDetector) analyzeOptimizedUserAgent(userAgent string) (float64, []string) {
	// Check cache first
	obd.cacheMu.RLock()
	if cached, exists := obd.patternCache[userAgent]; exists && time.Now().Before(obd.cacheExpiry) {
		obd.cacheMu.RUnlock()
		if cached {
			return 0.8, []string{"Cached bot user agent"}
		}
		return 0.0, []string{}
	}
	obd.cacheMu.RUnlock()
	
	score := 0.0
	reasons := make([]string, 0, 5)
	
	// Check for suspicious patterns using pre-compiled regexes
	for _, pattern := range obd.suspiciousPatterns {
		if pattern.MatchString(strings.ToLower(userAgent)) {
			score += 0.3
			reasons = append(reasons, "Suspicious user agent pattern")
		}
	}
	
	// Check for empty or very short user agents
	if len(userAgent) < 10 {
		score += 0.4
		reasons = append(reasons, "Very short user agent")
	}
	
	// Cache result
	obd.cacheMu.Lock()
	obd.patternCache[userAgent] = score > 0.5
	if time.Now().After(obd.cacheExpiry) {
		obd.patternCache = make(map[string]bool)
		obd.cacheExpiry = time.Now().Add(obd.cacheDuration)
	}
	obd.cacheMu.Unlock()
	
	return score, reasons
}

// analyzeOptimizedRequestFrequency analyzes request frequency with optimizations
func (obd *OptimizedBotDetector) analyzeOptimizedRequestFrequency(pattern *OptimizedRequestPattern) (float64, []string) {
	pattern.mu.RLock()
	defer pattern.mu.RUnlock()
	
	score := 0.0
	reasons := make([]string, 0, 3)
	
	if pattern.RequestCount < 2 {
		return score, reasons
	}
	
	// Calculate requests per minute with optimizations
	duration := pattern.LastRequest.Sub(pattern.FirstRequest)
	if duration > 0 {
		rpm := float64(pattern.RequestCount) / duration.Minutes()
		
		// Use pre-calculated thresholds
		if rpm > 60 {
			score += 0.4
			reasons = append(reasons, "High request frequency")
		} else if rpm > 30 {
			score += 0.2
			reasons = append(reasons, "Elevated request frequency")
		}
	}
	
	// Check for burst patterns with optimized calculation
	if len(pattern.ResponseTimes) > 10 {
		recent := pattern.ResponseTimes[len(pattern.ResponseTimes)-10:]
		avgInterval := obd.calculateOptimizedAverageInterval(recent)
		
		if avgInterval < 100*time.Millisecond {
			score += 0.3
			reasons = append(reasons, "Burst request pattern detected")
		}
	}
	
	return score, reasons
}

// analyzeOptimizedResponseTimes analyzes response times with optimizations
func (obd *OptimizedBotDetector) analyzeOptimizedResponseTimes(pattern *OptimizedRequestPattern) (float64, []string) {
	pattern.mu.RLock()
	defer pattern.mu.RUnlock()
	
	score := 0.0
	reasons := make([]string, 0, 2)
	
	if len(pattern.ResponseTimes) < 5 {
		return score, reasons
	}
	
	// Calculate average response time with optimizations
	var total time.Duration
	for _, rt := range pattern.ResponseTimes {
		total += rt
	}
	avgResponseTime := total / time.Duration(len(pattern.ResponseTimes))
	
	// Use pre-calculated thresholds
	if avgResponseTime < 50*time.Millisecond {
		score += 0.3
		reasons = append(reasons, "Unusually fast response times")
	}
	
	// Check for very consistent response times
	if len(pattern.ResponseTimes) > 10 {
		variance := obd.calculateOptimizedVariance(pattern.ResponseTimes)
		if variance < 10*time.Millisecond {
			score += 0.2
			reasons = append(reasons, "Very consistent response times")
		}
	}
	
	return score, reasons
}

// analyzeOptimizedErrorPatterns analyzes error patterns with optimizations
func (obd *OptimizedBotDetector) analyzeOptimizedErrorPatterns(pattern *OptimizedRequestPattern) (float64, []string) {
	pattern.mu.RLock()
	defer pattern.mu.RUnlock()
	
	score := 0.0
	reasons := make([]string, 0, 2)
	
	if pattern.RequestCount == 0 {
		return score, reasons
	}
	
	errorRate := float64(pattern.ErrorCount) / float64(pattern.RequestCount)
	
	// Use pre-calculated thresholds
	if errorRate > 0.5 {
		score += 0.4
		reasons = append(reasons, "High error rate")
	} else if errorRate > 0.2 {
		score += 0.2
		reasons = append(reasons, "Elevated error rate")
	}
	
	return score, reasons
}

// analyzeOptimizedPathPatterns analyzes path patterns with optimizations
func (obd *OptimizedBotDetector) analyzeOptimizedPathPatterns(path string, pattern *OptimizedRequestPattern) (float64, []string) {
	score := 0.0
	reasons := make([]string, 0, 3)
	
	// Pre-defined suspicious paths for performance
	suspiciousPaths := []string{
		"/admin", "/wp-admin", "/phpmyadmin", "/.env",
		"/config", "/backup", "/test", "/debug",
	}
	
	for _, suspiciousPath := range suspiciousPaths {
		if strings.Contains(path, suspiciousPath) {
			score += 0.3
			reasons = append(reasons, "Suspicious path accessed")
		}
	}
	
	// Check for repetitive path access with optimizations
	pattern.mu.RLock()
	if len(pattern.RequestPaths) > 0 {
		totalPaths := 0
		for _, count := range pattern.RequestPaths {
			totalPaths += count
		}
		
		// If one path is accessed more than 80% of the time
		for pathName, count := range pattern.RequestPaths {
			if float64(count)/float64(totalPaths) > 0.8 {
				score += 0.2
				reasons = append(reasons, "Repetitive path access")
			}
		}
	}
	pattern.mu.RUnlock()
	
	return score, reasons
}

// calculateOptimizedConfidence calculates confidence with optimizations
func (obd *OptimizedBotDetector) calculateOptimizedConfidence(pattern *OptimizedRequestPattern) float64 {
	pattern.mu.RLock()
	defer pattern.mu.RUnlock()
	
	// More data points = higher confidence
	dataPoints := pattern.RequestCount
	
	// Use pre-calculated thresholds
	if dataPoints < 5 {
		return 0.3
	} else if dataPoints < 20 {
		return 0.6
	} else {
		return 0.9
	}
}

// calculateOptimizedAverageInterval calculates average interval with optimizations
func (obd *OptimizedBotDetector) calculateOptimizedAverageInterval(times []time.Duration) time.Duration {
	if len(times) < 2 {
		return 0
	}
	
	var total time.Duration
	for i := 1; i < len(times); i++ {
		total += times[i] - times[i-1]
	}
	
	return total / time.Duration(len(times)-1)
}

// calculateOptimizedVariance calculates variance with optimizations
func (obd *OptimizedBotDetector) calculateOptimizedVariance(times []time.Duration) time.Duration {
	if len(times) < 2 {
		return 0
	}
	
	// Calculate mean
	var total time.Duration
	for _, t := range times {
		total += t
	}
	mean := total / time.Duration(len(times))
	
	// Calculate variance
	var variance time.Duration
	for _, t := range times {
		diff := t - mean
		if diff < 0 {
			diff = -diff
		}
		variance += diff * diff
	}
	
	return variance / time.Duration(len(times))
}

// initializeOptimizedPatterns initializes patterns with pre-compiled regexes
func (obd *OptimizedBotDetector) initializeOptimizedPatterns() {
	patterns := []string{
		`headless`,
		`phantom`,
		`selenium`,
		`webdriver`,
		`automation`,
		`bot`,
		`crawler`,
		`spider`,
		`scraper`,
		`curl`,
		`wget`,
		`python-requests`,
		`go-http-client`,
		`java/`,
		`okhttp`,
	}
	
	for _, pattern := range patterns {
		regex, err := regexp.Compile(`(?i)` + pattern)
		if err == nil {
			obd.suspiciousPatterns = append(obd.suspiciousPatterns, regex)
		}
	}
}

// CleanupExpiredPatterns removes old patterns with optimizations
func (obd *OptimizedBotDetector) CleanupExpiredPatterns() {
	obd.mu.Lock()
	defer obd.mu.Unlock()
	
	now := time.Now()
	for ip, pattern := range obd.requestPatterns {
		// Remove patterns older than 1 hour
		if now.Sub(pattern.LastRequest) > time.Hour {
			delete(obd.requestPatterns, ip)
		}
	}
	
	// Clean up user agent cache
	for ua, stats := range obd.userAgents {
		if now.Sub(stats.LastSeen) > time.Hour {
			delete(obd.userAgents, ua)
		}
	}
}

// GetStats returns optimized bot detector statistics
func (obd *OptimizedBotDetector) GetStats() map[string]interface{} {
	obd.mu.RLock()
	defer obd.mu.RUnlock()
	
	obd.cacheMu.RLock()
	cacheSize := len(obd.patternCache)
	obd.cacheMu.RUnlock()
	
	return map[string]interface{}{
		"tracked_ips":        len(obd.requestPatterns),
		"suspicious_patterns": len(obd.suspiciousPatterns),
		"unique_user_agents": len(obd.userAgents),
		"pattern_cache_size": cacheSize,
		"cache_expiry":       obd.cacheExpiry,
	}
}
