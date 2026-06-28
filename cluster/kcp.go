package cluster

import (
	"fmt"

	"github.com/lonng/nano/internal/log"
	kcp "github.com/xtaci/kcp-go"
)

const (
	defaultKCPInterval = 20
	defaultKCPSndWnd   = 128
	defaultKCPRcvWnd   = 512
)

// KCPConfig describes optional KCP transport tuning for client connections.
type KCPConfig struct {
	NoDelay     bool
	Interval    int
	Resend      int
	NC          bool
	MTU         int
	SndWnd      int
	RcvWnd      int
	ReadBuffer  int
	WriteBuffer int
	DSCP        int
}

func normalizeKCPConfig(config KCPConfig) KCPConfig {
	if config.Interval <= 0 {
		config.Interval = defaultKCPInterval
	}
	if config.SndWnd <= 0 {
		config.SndWnd = defaultKCPSndWnd
	}
	if config.RcvWnd <= 0 {
		config.RcvWnd = defaultKCPRcvWnd
	}
	return config
}

func (n *Node) listenAndServeKCP() {
	config := normalizeKCPConfig(n.KCPConfig)
	n.logStartup("kcp_listen_begin", "addr", n.KCPAddr)
	listener, err := kcp.ListenWithOptions(n.KCPAddr, nil, 0, 0)
	if err != nil {
		log.Fatalf("Nano startup kcp_listen_failed role=%s service=%s kcp=%s error=%v", n.role(), n.ServiceAddr, n.KCPAddr, err)
	}
	n.logStartup("kcp_listen_success",
		"addr", n.KCPAddr,
		"nodelay", config.NoDelay,
		"interval", config.Interval,
		"resend", config.Resend,
		"nc", config.NC,
		"mtu", config.MTU,
		"sndwnd", config.SndWnd,
		"rcvwnd", config.RcvWnd,
	)

	defer listener.Close()
	if config.ReadBuffer > 0 {
		if err := listener.SetReadBuffer(config.ReadBuffer); err != nil {
			log.Println(fmt.Sprintf("Set KCP read buffer failed: %s", err.Error()))
		}
	}
	if config.WriteBuffer > 0 {
		if err := listener.SetWriteBuffer(config.WriteBuffer); err != nil {
			log.Println(fmt.Sprintf("Set KCP write buffer failed: %s", err.Error()))
		}
	}
	if config.DSCP > 0 {
		if err := listener.SetDSCP(config.DSCP); err != nil {
			log.Println(fmt.Sprintf("Set KCP DSCP failed: %s", err.Error()))
		}
	}

	for {
		conn, err := listener.AcceptKCP()
		if err != nil {
			log.Println("Nano kcp accept failed", "addr", n.KCPAddr, "error", err)
			continue
		}

		conn.SetNoDelay(boolToInt(config.NoDelay), config.Interval, config.Resend, boolToInt(config.NC))
		conn.SetWindowSize(config.SndWnd, config.RcvWnd)
		if config.MTU > 0 && !conn.SetMtu(config.MTU) {
			log.Println(fmt.Sprintf("Set KCP MTU failed: %d", config.MTU))
		}

		go n.handler.handle(conn)
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
