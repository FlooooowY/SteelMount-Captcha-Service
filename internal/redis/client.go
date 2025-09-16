package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/FlooooowY/SteelMount-Captcha-Service/internal/config"
	"github.com/go-redis/redis/v8"
)

// Client wraps Redis client with configuration
type Client struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewClient creates a new Redis client
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("redis config cannot be nil")
	}

	// Parse Redis URL
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Set configuration options
	opt.PoolSize = cfg.PoolSize
	opt.MinIdleConns = cfg.MinIdleConns
	opt.MaxRetries = cfg.MaxRetries
	opt.DialTimeout = cfg.DialTimeout
	opt.ReadTimeout = cfg.ReadTimeout
	opt.WriteTimeout = cfg.WriteTimeout

	// Create Redis client
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// GetClient returns the underlying Redis client
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.client.Close()
}

// Health checks Redis connection health
func (c *Client) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// GetStats returns Redis client statistics
func (c *Client) GetStats() map[string]interface{} {
	stats := c.client.PoolStats()
	
	return map[string]interface{}{
		"hits":           stats.Hits,
		"misses":         stats.Misses,
		"timeouts":       stats.Timeouts,
		"total_conns":    stats.TotalConns,
		"idle_conns":     stats.IdleConns,
		"stale_conns":    stats.StaleConns,
		"pool_size":      c.config.PoolSize,
		"min_idle_conns": c.config.MinIdleConns,
	}
}
