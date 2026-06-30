package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type Stats struct {
	started       int64
	connected     int64
	loggedIn      int64
	roleCreated   int64
	pluginSuccess int64
	failed        int64
}

func (s *Stats) IncStarted()       { atomic.AddInt64(&s.started, 1) }
func (s *Stats) IncConnected()     { atomic.AddInt64(&s.connected, 1) }
func (s *Stats) IncLoggedIn()      { atomic.AddInt64(&s.loggedIn, 1) }
func (s *Stats) IncRoleCreated()   { atomic.AddInt64(&s.roleCreated, 1) }
func (s *Stats) IncPluginSuccess() { atomic.AddInt64(&s.pluginSuccess, 1) }
func (s *Stats) IncFailed()        { atomic.AddInt64(&s.failed, 1) }

func (s *Stats) Snapshot() StatsSnapshot {
	return StatsSnapshot{
		Started:       atomic.LoadInt64(&s.started),
		Connected:     atomic.LoadInt64(&s.connected),
		LoggedIn:      atomic.LoadInt64(&s.loggedIn),
		RoleCreated:   atomic.LoadInt64(&s.roleCreated),
		PluginSuccess: atomic.LoadInt64(&s.pluginSuccess),
		Failed:        atomic.LoadInt64(&s.failed),
	}
}

type StatsSnapshot struct {
	Started       int64
	Connected     int64
	LoggedIn      int64
	RoleCreated   int64
	PluginSuccess int64
	Failed        int64
}

func (s StatsSnapshot) String() string {
	return fmt.Sprintf("started=%d connected=%d logged_in=%d role_created=%d plugin_success=%d failed=%d",
		s.Started, s.Connected, s.LoggedIn, s.RoleCreated, s.PluginSuccess, s.Failed)
}

func reportStats(done <-chan struct{}, stats *Stats, interval time.Duration, logger *RunLogger) {
	if interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Infof("stats %s", stats.Snapshot().String())
		case <-done:
			return
		}
	}
}
