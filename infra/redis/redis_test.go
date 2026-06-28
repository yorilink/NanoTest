package redis

import (
	"testing"
	"time"
)

func TestNewAppliesConfig(t *testing.T) {
	cfg := Config{
		Addr:         "127.0.0.1:6380",
		Password:     "secret",
		DB:           2,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  600 * time.Millisecond,
		WriteTimeout: 700 * time.Millisecond,
		PoolSize:     12,
		MinIdleConns: 3,
	}

	client := New(cfg)
	defer client.Close()

	opt := client.Raw().Options()
	if opt.Addr != cfg.Addr {
		t.Fatalf("addr = %q, want %q", opt.Addr, cfg.Addr)
	}
	if opt.Password != cfg.Password {
		t.Fatalf("password = %q, want %q", opt.Password, cfg.Password)
	}
	if opt.DB != cfg.DB {
		t.Fatalf("db = %d, want %d", opt.DB, cfg.DB)
	}
	if opt.DialTimeout != cfg.DialTimeout {
		t.Fatalf("dial timeout = %s, want %s", opt.DialTimeout, cfg.DialTimeout)
	}
	if opt.ReadTimeout != cfg.ReadTimeout {
		t.Fatalf("read timeout = %s, want %s", opt.ReadTimeout, cfg.ReadTimeout)
	}
	if opt.WriteTimeout != cfg.WriteTimeout {
		t.Fatalf("write timeout = %s, want %s", opt.WriteTimeout, cfg.WriteTimeout)
	}
	if opt.PoolSize != cfg.PoolSize {
		t.Fatalf("pool size = %d, want %d", opt.PoolSize, cfg.PoolSize)
	}
	if opt.MinIdleConns != cfg.MinIdleConns {
		t.Fatalf("min idle conns = %d, want %d", opt.MinIdleConns, cfg.MinIdleConns)
	}
}
