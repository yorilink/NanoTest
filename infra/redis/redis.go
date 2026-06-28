package redis

import (
	"time"

	goredis "github.com/go-redis/redis"
)

// Config describes a Redis client.
type Config struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
}

// Client is a lightweight wrapper around go-redis.
type Client struct {
	client *goredis.Client
}

// New creates a redis client.
func New(cfg Config) *Client {
	opt := &goredis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	}
	return &Client{client: goredis.NewClient(opt)}
}

// Raw returns the underlying go-redis client.
func (c *Client) Raw() *goredis.Client {
	return c.client
}

// Ping checks whether the server is reachable.
func (c *Client) Ping() error {
	return c.client.Ping().Err()
}

// Get returns the value of a key.
func (c *Client) Get(key string) (string, error) {
	return c.client.Get(key).Result()
}

// Set sets a key to a value.
func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(key, value, expiration).Err()
}

// Del deletes keys.
func (c *Client) Del(keys ...string) error {
	return c.client.Del(keys...).Err()
}

// Close closes the client.
func (c *Client) Close() error {
	return c.client.Close()
}
