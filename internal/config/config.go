package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Redis      RedisConfig      `yaml:"redis"`
	Captcha    CaptchaConfig    `yaml:"captcha"`
	Security   SecurityConfig   `yaml:"security"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Balancer   BalancerConfig   `yaml:"balancer"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	MinPort         int           `yaml:"min_port"`
	MaxPort         int           `yaml:"max_port"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	StartupTimeout  time.Duration `yaml:"startup_timeout"`
	InitTimeout     time.Duration `yaml:"init_timeout"`
}

// RedisConfig contains Redis-related configuration
type RedisConfig struct {
	URL          string        `yaml:"url"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	MaxRetries   int           `yaml:"max_retries"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// CaptchaConfig contains captcha-related configuration
type CaptchaConfig struct {
	MaxActiveChallenges int            `yaml:"max_active_challenges"`
	MemoryLimitGB       int            `yaml:"memory_limit_gb"`
	TargetRPS           int            `yaml:"target_rps"`
	ChallengeTimeout    time.Duration  `yaml:"challenge_timeout"`
	CleanupInterval     time.Duration  `yaml:"cleanup_interval"`
	DragDrop            DragDropConfig `yaml:"drag_drop"`
	Click               ClickConfig    `yaml:"click"`
	Swipe               SwipeConfig    `yaml:"swipe"`
}

// DragDropConfig contains drag & drop captcha settings
type DragDropConfig struct {
	MinObjects   int `yaml:"min_objects"`
	MaxObjects   int `yaml:"max_objects"`
	CanvasWidth  int `yaml:"canvas_width"`
	CanvasHeight int `yaml:"canvas_height"`
}

// ClickConfig contains click captcha settings
type ClickConfig struct {
	MinClicks   int `yaml:"min_clicks"`
	MaxClicks   int `yaml:"max_clicks"`
	ClickRadius int `yaml:"click_radius"`
}

// SwipeConfig contains swipe captcha settings
type SwipeConfig struct {
	MinSwipes      int `yaml:"min_swipes"`
	MaxSwipes      int `yaml:"max_swipes"`
	SwipeThreshold int `yaml:"swipe_threshold"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	IPBlocking   IPBlockingConfig   `yaml:"ip_blocking"`
	BotDetection BotDetectionConfig `yaml:"bot_detection"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int           `yaml:"requests_per_minute"`
	BurstSize         int           `yaml:"burst_size"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
}

// IPBlockingConfig contains IP blocking settings
type IPBlockingConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MaxFailedAttempts int           `yaml:"max_failed_attempts"`
	BlockDuration     time.Duration `yaml:"block_duration"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval"`
}

// BotDetectionConfig contains bot detection settings
type BotDetectionConfig struct {
	Enabled            bool     `yaml:"enabled"`
	SuspiciousPatterns []string `yaml:"suspicious_patterns"`
}

// MonitoringConfig contains monitoring-related configuration
type MonitoringConfig struct {
	PrometheusPort  int           `yaml:"prometheus_port"`
	MetricsPath     string        `yaml:"metrics_path"`
	HealthCheckPath string        `yaml:"health_check_path"`
	Logging         LoggingConfig `yaml:"logging"`
	Tracing         TracingConfig `yaml:"tracing"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// TracingConfig contains tracing settings
type TracingConfig struct {
	Enabled        bool   `yaml:"enabled"`
	JaegerEndpoint string `yaml:"jaeger_endpoint"`
}

// BalancerConfig contains balancer-related configuration
type BalancerConfig struct {
	URL                  string        `yaml:"url"`
	Enabled              bool          `yaml:"enabled"`
	RegistrationInterval time.Duration `yaml:"registration_interval"`
	HeartbeatTimeout     time.Duration `yaml:"heartbeat_timeout"`
	MaxRetryAttempts     int           `yaml:"max_retry_attempts"`
	RetryDelay           time.Duration `yaml:"retry_delay"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	// Load from YAML file if it exists
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	overrideWithEnv(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromFile loads configuration from YAML file
func loadFromFile(config *Config, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// overrideWithEnv overrides configuration with environment variables
func overrideWithEnv(config *Config) {
	// Server configuration
	if minPort := os.Getenv("MIN_PORT"); minPort != "" {
		if port, err := strconv.Atoi(minPort); err == nil {
			config.Server.MinPort = port
		}
	}
	if maxPort := os.Getenv("MAX_PORT"); maxPort != "" {
		if port, err := strconv.Atoi(maxPort); err == nil {
			config.Server.MaxPort = port
		}
	}

	// Redis configuration
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		config.Redis.URL = redisURL
	}

	// Logging configuration
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Monitoring.Logging.Level = logLevel
	}

	// Metrics port
	if metricsPort := os.Getenv("METRICS_PORT"); metricsPort != "" {
		if port, err := strconv.Atoi(metricsPort); err == nil {
			config.Monitoring.PrometheusPort = port
		}
	}

	// Balancer configuration
	if balancerURL := os.Getenv("BALANCER_URL"); balancerURL != "" {
		config.Balancer.URL = balancerURL
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.MinPort <= 0 || config.Server.MaxPort <= 0 {
		return fmt.Errorf("invalid port range: min=%d, max=%d", config.Server.MinPort, config.Server.MaxPort)
	}
	if config.Server.MinPort >= config.Server.MaxPort {
		return fmt.Errorf("min port must be less than max port: min=%d, max=%d", config.Server.MinPort, config.Server.MaxPort)
	}

	// Validate captcha configuration
	if config.Captcha.MaxActiveChallenges <= 0 {
		return fmt.Errorf("max active challenges must be positive: %d", config.Captcha.MaxActiveChallenges)
	}
	if config.Captcha.MemoryLimitGB <= 0 {
		return fmt.Errorf("memory limit must be positive: %d", config.Captcha.MemoryLimitGB)
	}
	if config.Captcha.TargetRPS <= 0 {
		return fmt.Errorf("target RPS must be positive: %d", config.Captcha.TargetRPS)
	}

	// Validate Redis configuration
	if config.Redis.URL == "" {
		return fmt.Errorf("redis URL is required")
	}

	return nil
}
