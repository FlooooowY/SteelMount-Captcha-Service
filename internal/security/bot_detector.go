package security

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// BotDetector detects bot behavior and suspicious patterns
type BotDetector struct {
	mu                  sync.RWMutex
	suspiciousPatterns  []*regexp.Regexp
	behaviorPatterns    map[string]*BehaviorPattern
	userAgents         map[string]int
	requestPatterns    map[string]*RequestPattern
}

// BehaviorPattern represents a behavior pattern to detect
type BehaviorPattern struct {
	Name        string
	Pattern     *regexp.Regexp
	Weight      float64
	Description string
}

// RequestPattern tracks request patterns for an IP
type RequestPattern struct {
	IP              string
	RequestCount    int
	FirstRequest    time.Time
	LastRequest     time.Time
	UserAgents      map[string]int
	RequestPaths    map[string]int
	ResponseTimes   []time.Duration
	ErrorCount      int
}

// BotScore represents a bot detection score
type BotScore struct {
	IP          string
	Score       float64
	Confidence  float64
	Reasons     []string
	Timestamp   time.Time
}

// NewBotDetector creates a new bot detector
func NewBotDetector() *BotDetector {
	bd := &BotDetector{
		behaviorPatterns: make(map[string]*BehaviorPattern),
		userAgents:      make(map[string]int),
		requestPatterns: make(map[string]*RequestPattern),
	}
	
	// Initialize suspicious patterns
	bd.initializeSuspiciousPatterns()
	
	return bd
}

// AnalyzeRequest analyzes a request for bot behavior
func (bd *BotDetector) AnalyzeRequest(ctx context.Context, ip string, userAgent string, path string, responseTime time.Duration, isError bool) (*BotScore, error) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	// Get or create request pattern for IP
	pattern, exists := bd.requestPatterns[ip]
	if !exists {
		pattern = &RequestPattern{
			IP:           ip,
			UserAgents:   make(map[string]int),
			RequestPaths: make(map[string]int),
			ResponseTimes: make([]time.Duration, 0),
		}
		bd.requestPatterns[ip] = pattern
	}
	
	// Update pattern
	now := time.Now()
	if pattern.FirstRequest.IsZero() {
		pattern.FirstRequest = now
	}
	pattern.LastRequest = now
	pattern.RequestCount++
	pattern.UserAgents[userAgent]++
	pattern.RequestPaths[path]++
	pattern.ResponseTimes = append(pattern.ResponseTimes, responseTime)
	if isError {
		pattern.ErrorCount++
	}
	
	// Keep only last 100 response times
	if len(pattern.ResponseTimes) > 100 {
		pattern.ResponseTimes = pattern.ResponseTimes[len(pattern.ResponseTimes)-100:]
	}
	
	// Calculate bot score
	score := bd.calculateBotScore(ip, userAgent, path, pattern)
	
	return score, nil
}

// calculateBotScore calculates the bot probability score
func (bd *BotDetector) calculateBotScore(ip string, userAgent string, path string, pattern *RequestPattern) *BotScore {
	score := 0.0
	reasons := []string{}
	
	// Check user agent patterns
	uaScore, uaReasons := bd.analyzeUserAgent(userAgent)
	score += uaScore
	reasons = append(reasons, uaReasons...)
	
	// Check request frequency
	freqScore, freqReasons := bd.analyzeRequestFrequency(pattern)
	score += freqScore
	reasons = append(reasons, freqReasons...)
	
	// Check response time patterns
	timeScore, timeReasons := bd.analyzeResponseTimes(pattern)
	score += timeScore
	reasons = append(reasons, timeReasons...)
	
	// Check error patterns
	errorScore, errorReasons := bd.analyzeErrorPatterns(pattern)
	score += errorScore
	reasons = append(reasons, errorReasons...)
	
	// Check path patterns
	pathScore, pathReasons := bd.analyzePathPatterns(path, pattern)
	score += pathScore
	reasons = append(reasons, pathReasons...)
	
	// Calculate confidence based on data points
	confidence := bd.calculateConfidence(pattern)
	
	return &BotScore{
		IP:         ip,
		Score:      score,
		Confidence: confidence,
		Reasons:    reasons,
		Timestamp:  time.Now(),
	}
}

// analyzeUserAgent analyzes user agent for bot patterns
func (bd *BotDetector) analyzeUserAgent(userAgent string) (float64, []string) {
	score := 0.0
	reasons := []string{}
	
	// Check for suspicious patterns
	for _, pattern := range bd.suspiciousPatterns {
		if pattern.MatchString(strings.ToLower(userAgent)) {
			score += 0.3
			reasons = append(reasons, fmt.Sprintf("Suspicious user agent pattern: %s", pattern.String()))
		}
	}
	
	// Check for empty or very short user agents
	if len(userAgent) < 10 {
		score += 0.4
		reasons = append(reasons, "Very short user agent")
	}
	
	// Check for common bot user agents
	botPatterns := []string{
		"bot", "crawler", "spider", "scraper", "headless",
		"selenium", "phantom", "chrome-lighthouse",
		"googlebot", "bingbot", "slurp", "duckduckbot",
	}
	
	for _, botPattern := range botPatterns {
		if strings.Contains(strings.ToLower(userAgent), botPattern) {
			score += 0.2
			reasons = append(reasons, fmt.Sprintf("Bot user agent detected: %s", botPattern))
		}
	}
	
	return score, reasons
}

// analyzeRequestFrequency analyzes request frequency patterns
func (bd *BotDetector) analyzeRequestFrequency(pattern *RequestPattern) (float64, []string) {
	score := 0.0
	reasons := []string{}
	
	if pattern.RequestCount < 2 {
		return score, reasons
	}
	
	// Calculate requests per minute
	duration := pattern.LastRequest.Sub(pattern.FirstRequest)
	if duration > 0 {
		rpm := float64(pattern.RequestCount) / duration.Minutes()
		
		// High frequency requests
		if rpm > 60 { // More than 1 request per second
			score += 0.4
			reasons = append(reasons, fmt.Sprintf("High request frequency: %.2f req/min", rpm))
		} else if rpm > 30 { // More than 0.5 requests per second
			score += 0.2
			reasons = append(reasons, fmt.Sprintf("Elevated request frequency: %.2f req/min", rpm))
		}
	}
	
	// Check for burst patterns
	if len(pattern.ResponseTimes) > 10 {
		recent := pattern.ResponseTimes[len(pattern.ResponseTimes)-10:]
		avgInterval := bd.calculateAverageInterval(recent)
		
		if avgInterval < 100*time.Millisecond {
			score += 0.3
			reasons = append(reasons, "Burst request pattern detected")
		}
	}
	
	return score, reasons
}

// analyzeResponseTimes analyzes response time patterns
func (bd *BotDetector) analyzeResponseTimes(pattern *RequestPattern) (float64, []string) {
	score := 0.0
	reasons := []string{}
	
	if len(pattern.ResponseTimes) < 5 {
		return score, reasons
	}
	
	// Calculate average response time
	var total time.Duration
	for _, rt := range pattern.ResponseTimes {
		total += rt
	}
	avgResponseTime := total / time.Duration(len(pattern.ResponseTimes))
	
	// Very fast response times might indicate automation
	if avgResponseTime < 50*time.Millisecond {
		score += 0.3
		reasons = append(reasons, "Unusually fast response times")
	}
	
	// Check for very consistent response times (might indicate automation)
	if len(pattern.ResponseTimes) > 10 {
		variance := bd.calculateVariance(pattern.ResponseTimes)
		if variance < 10*time.Millisecond {
			score += 0.2
			reasons = append(reasons, "Very consistent response times")
		}
	}
	
	return score, reasons
}

// analyzeErrorPatterns analyzes error patterns
func (bd *BotDetector) analyzeErrorPatterns(pattern *RequestPattern) (float64, []string) {
	score := 0.0
	reasons := []string{}
	
	if pattern.RequestCount == 0 {
		return score, reasons
	}
	
	errorRate := float64(pattern.ErrorCount) / float64(pattern.RequestCount)
	
	// High error rate might indicate bot behavior
	if errorRate > 0.5 {
		score += 0.4
		reasons = append(reasons, fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
	} else if errorRate > 0.2 {
		score += 0.2
		reasons = append(reasons, fmt.Sprintf("Elevated error rate: %.2f%%", errorRate*100))
	}
	
	return score, reasons
}

// analyzePathPatterns analyzes request path patterns
func (bd *BotDetector) analyzePathPatterns(path string, pattern *RequestPattern) (float64, []string) {
	score := 0.0
	reasons := []string{}
	
	// Check for suspicious paths
	suspiciousPaths := []string{
		"/admin", "/wp-admin", "/phpmyadmin", "/.env",
		"/config", "/backup", "/test", "/debug",
	}
	
	for _, suspiciousPath := range suspiciousPaths {
		if strings.Contains(path, suspiciousPath) {
			score += 0.3
			reasons = append(reasons, fmt.Sprintf("Suspicious path accessed: %s", suspiciousPath))
		}
	}
	
	// Check for repetitive path access
	if len(pattern.RequestPaths) > 0 {
		totalPaths := 0
		for _, count := range pattern.RequestPaths {
			totalPaths += count
		}
		
		// If one path is accessed more than 80% of the time
		for pathName, count := range pattern.RequestPaths {
			if float64(count)/float64(totalPaths) > 0.8 {
				score += 0.2
				reasons = append(reasons, fmt.Sprintf("Repetitive path access: %s", pathName))
			}
		}
	}
	
	return score, reasons
}

// calculateConfidence calculates confidence in the bot score
func (bd *BotDetector) calculateConfidence(pattern *RequestPattern) float64 {
	// More data points = higher confidence
	dataPoints := pattern.RequestCount
	
	if dataPoints < 5 {
		return 0.3
	} else if dataPoints < 20 {
		return 0.6
	} else {
		return 0.9
	}
}

// calculateAverageInterval calculates average interval between requests
func (bd *BotDetector) calculateAverageInterval(times []time.Duration) time.Duration {
	if len(times) < 2 {
		return 0
	}
	
	var total time.Duration
	for i := 1; i < len(times); i++ {
		total += times[i] - times[i-1]
	}
	
	return total / time.Duration(len(times)-1)
}

// calculateVariance calculates variance in response times
func (bd *BotDetector) calculateVariance(times []time.Duration) time.Duration {
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

// initializeSuspiciousPatterns initializes suspicious user agent patterns
func (bd *BotDetector) initializeSuspiciousPatterns() {
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
			bd.suspiciousPatterns = append(bd.suspiciousPatterns, regex)
		}
	}
}

// CleanupExpiredPatterns removes old request patterns
func (bd *BotDetector) CleanupExpiredPatterns() {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	now := time.Now()
	for ip, pattern := range bd.requestPatterns {
		// Remove patterns older than 1 hour
		if now.Sub(pattern.LastRequest) > time.Hour {
			delete(bd.requestPatterns, ip)
		}
	}
}

// GetStats returns bot detector statistics
func (bd *BotDetector) GetStats() map[string]interface{} {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	return map[string]interface{}{
		"tracked_ips":        len(bd.requestPatterns),
		"suspicious_patterns": len(bd.suspiciousPatterns),
		"unique_user_agents": len(bd.userAgents),
	}
}
