package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Addr         string
	Path         string
	TLS          bool
	Count        int
	Concurrency  int
	StartAccount int64
	Timeout      time.Duration
	KeepAlive    time.Duration
	ReportEvery  time.Duration
	LogDir       string
}

func main() {
	cfg := parseConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	targetURL, err := buildURL(cfg.Addr, cfg.Path, cfg.TLS)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	logger, err := NewRunLogger(cfg.LogDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	exitCode := 0
	defer func() {
		_ = logger.Close()
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()
	logger.Infof("log_file=%s", logger.Path())
	logger.Infof("target=%s count=%d concurrency=%d start_account=%d timeout=%s keepalive=%s",
		targetURL, cfg.Count, cfg.Concurrency, cfg.StartAccount, cfg.Timeout, cfg.KeepAlive)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	stats := &Stats{}
	doneReport := make(chan struct{})
	go reportStats(doneReport, stats, cfg.ReportEvery, logger)

	start := time.Now()
	plugins := buildPlugins(cfg)
	err = run(ctx, cfg, targetURL, plugins, stats, logger)
	close(doneReport)

	logger.Infof("final %s elapsed=%s", stats.Snapshot().String(), time.Since(start).String())
	if err != nil {
		logger.Errorf("run failed: %v", err)
		exitCode = 1
	}
}

func parseConfig() Config {
	cfg := Config{}
	flag.StringVar(&cfg.Addr, "addr", "127.0.0.1:34590", "gate WebSocket address")
	flag.StringVar(&cfg.Path, "path", "/nano", "gate WebSocket path")
	flag.BoolVar(&cfg.TLS, "tls", false, "use wss")
	flag.IntVar(&cfg.Count, "count", 1, "robot count")
	flag.IntVar(&cfg.Concurrency, "concurrency", 1, "max concurrent login workers")
	flag.Int64Var(&cfg.StartAccount, "start-account", 1, "first demo account id")
	flag.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "single network operation timeout")
	flag.DurationVar(&cfg.KeepAlive, "keepalive", 0, "hold successful connections before closing")
	flag.DurationVar(&cfg.ReportEvery, "report", 5*time.Second, "stats report interval, 0 disables periodic reports")
	flag.StringVar(&cfg.LogDir, "log-dir", "robot/logs", "robot log directory")
	flag.Parse()
	return cfg
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("-addr cannot be empty")
	}
	if c.Count < 1 {
		return fmt.Errorf("-count must be greater than 0")
	}
	if c.Concurrency < 1 {
		return fmt.Errorf("-concurrency must be greater than 0")
	}
	if c.Concurrency > c.Count {
		c.Concurrency = c.Count
	}
	if c.StartAccount < 1 {
		return fmt.Errorf("-start-account must be greater than 0")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("-timeout must be greater than 0")
	}
	return nil
}

func run(ctx context.Context, cfg Config, targetURL string, plugins []Plugin, stats *Stats, logger *RunLogger) error {
	jobs := make(chan int64)
	var wg sync.WaitGroup

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for accountID := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				runRobot(ctx, cfg, targetURL, accountID, plugins, stats, logger)
			}
		}()
	}

	for i := 0; i < cfg.Count; i++ {
		accountID := cfg.StartAccount + int64(i)
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		case jobs <- accountID:
		}
	}

	close(jobs)
	wg.Wait()

	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func runRobot(ctx context.Context, cfg Config, targetURL string, accountID int64, plugins []Plugin, stats *Stats, logger *RunLogger) {
	stats.IncStarted()

	client := NewClient(cfg.Timeout)
	defer client.Close()

	opCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	_, err := client.Connect(opCtx, targetURL)
	cancel()
	if err != nil {
		stats.IncFailed()
		logger.Errorf("account=%d stage=connect failed: %v", accountID, err)
		return
	}
	stats.IncConnected()

	token := fmt.Sprintf("demo:%d", accountID)
	opCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
	login, err := client.Login(opCtx, token)
	cancel()
	if err != nil {
		stats.IncFailed()
		logger.Errorf("account=%d stage=login failed: %v", accountID, err)
		return
	}

	if login.NeedCreateRole {
		opCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		login, err = client.CreateRole(opCtx, token, fmt.Sprintf("robot-%d", accountID))
		cancel()
		if err != nil {
			stats.IncFailed()
			logger.Errorf("account=%d stage=create_role failed: %v", accountID, err)
			return
		}
		stats.IncRoleCreated()
	}

	stats.IncLoggedIn()

	for _, plugin := range plugins {
		opCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		err = plugin.Run(opCtx, client, accountID, login)
		cancel()
		if err != nil {
			stats.IncFailed()
			logger.Errorf("account=%d stage=plugin plugin=%s failed: %v", accountID, plugin.Name(), err)
			return
		}
		stats.IncPluginSuccess()
	}

	if cfg.KeepAlive > 0 {
		select {
		case <-time.After(cfg.KeepAlive):
		case <-ctx.Done():
		}
	}
}
