package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/internal/codec"
	"github.com/lonng/nano/internal/message"
	"github.com/lonng/nano/internal/packet"
)

const (
	routeGateLogin      = "GateService.Login"
	routeGateCreateRole = "GateService.CreateRole"
)

var (
	handshakePacket, _    = codec.Encode(packet.Handshake, nil)
	handshakeAckPacket, _ = codec.Encode(packet.HandshakeAck, nil)
	heartbeatPacket, _    = codec.Encode(packet.Heartbeat, nil)
)

type Client struct {
	conn    *websocket.Conn
	dec     *codec.Decoder
	timeout time.Duration
	readTTL time.Duration

	mu        sync.Mutex
	writeMu   sync.Mutex
	nextID    uint64
	responses map[uint64]chan response
	closed    chan struct{}
	closeOnce sync.Once
}

type response struct {
	data []byte
	err  error
}

type HandshakeResponse struct {
	Code int `json:"code"`
	Sys  struct {
		Heartbeat  float64           `json:"heartbeat"`
		ServerTime int64             `json:"servertime"`
		Dict       map[string]uint16 `json:"dict"`
	} `json:"sys"`
}

type LoginRequest struct {
	Token string `json:"token"`
}

type CreateRoleRequest struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type PlayerSummary struct {
	AccountID      int64  `json:"accountId"`
	PlayerID       int64  `json:"playerId"`
	Name           string `json:"name"`
	GameServerAddr string `json:"gameServerAddr"`
}

type GateResponse struct {
	Code           int            `json:"code"`
	Message        string         `json:"message,omitempty"`
	NeedCreateRole bool           `json:"needCreateRole,omitempty"`
	RoleExisted    bool           `json:"roleExisted,omitempty"`
	Player         *PlayerSummary `json:"player,omitempty"`
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		dec:       codec.NewDecoder(),
		timeout:   timeout,
		nextID:    1,
		responses: make(map[uint64]chan response),
		closed:    make(chan struct{}),
	}
}

func (c *Client) Connect(ctx context.Context, rawURL string) (*HandshakeResponse, error) {
	dialer := websocket.Dialer{HandshakeTimeout: c.timeout}
	conn, _, err := dialer.DialContext(ctx, rawURL, nil)
	if err != nil {
		return nil, err
	}
	c.conn = conn

	if err := c.writePacket(handshakePacket); err != nil {
		c.Close()
		return nil, err
	}

	hs, err := c.readHandshake(ctx)
	if err != nil {
		c.Close()
		return nil, err
	}

	if err := c.writePacket(handshakeAckPacket); err != nil {
		c.Close()
		return nil, err
	}

	c.readTTL = 3 * c.timeout
	if hs.Sys.Heartbeat > 0 {
		heartbeatTTL := time.Duration(hs.Sys.Heartbeat*3) * time.Second
		if heartbeatTTL > c.readTTL {
			c.readTTL = heartbeatTTL
		}
	}
	go c.readLoop()
	return hs, nil
}

func (c *Client) Request(ctx context.Context, route string, payload interface{}, out interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id, ch := c.registerResponse()
	msg := &message.Message{
		Type:  message.Request,
		ID:    id,
		Route: route,
		Data:  data,
	}
	encoded, err := msg.Encode()
	if err != nil {
		c.removeResponse(id)
		return err
	}
	p, err := codec.Encode(packet.Data, encoded)
	if err != nil {
		c.removeResponse(id)
		return err
	}
	if err := c.writePacket(p); err != nil {
		c.removeResponse(id)
		return err
	}

	select {
	case rsp := <-ch:
		if rsp.err != nil {
			return rsp.err
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(rsp.data, out)
	case <-ctx.Done():
		c.removeResponse(id)
		return ctx.Err()
	case <-c.closed:
		c.removeResponse(id)
		return errors.New("connection closed")
	}
}

func (c *Client) Login(ctx context.Context, token string) (*GateResponse, error) {
	var rsp GateResponse
	if err := c.Request(ctx, routeGateLogin, &LoginRequest{Token: token}, &rsp); err != nil {
		return nil, err
	}
	if rsp.Code != 0 {
		return &rsp, fmt.Errorf("login failed: code=%d message=%s", rsp.Code, rsp.Message)
	}
	return &rsp, nil
}

func (c *Client) CreateRole(ctx context.Context, token, name string) (*GateResponse, error) {
	var rsp GateResponse
	req := &CreateRoleRequest{Token: token, Name: name}
	if err := c.Request(ctx, routeGateCreateRole, req, &rsp); err != nil {
		return nil, err
	}
	if rsp.Code != 0 {
		return &rsp, fmt.Errorf("create role failed: code=%d message=%s", rsp.Code, rsp.Message)
	}
	return &rsp, nil
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

func (c *Client) registerResponse() (uint64, chan response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID
	c.nextID++
	ch := make(chan response, 1)
	c.responses[id] = ch
	return id, ch
}

func (c *Client) removeResponse(id uint64) {
	c.mu.Lock()
	delete(c.responses, id)
	c.mu.Unlock()
}

func (c *Client) dispatchResponse(id uint64, rsp response) {
	c.mu.Lock()
	ch := c.responses[id]
	delete(c.responses, id)
	c.mu.Unlock()

	if ch != nil {
		ch <- rsp
	}
}

func (c *Client) writePacket(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.timeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	}
	return c.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *Client) readHandshake(ctx context.Context) (*HandshakeResponse, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if c.timeout > 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return nil, err
		}
		packets, err := c.dec.Decode(data)
		if err != nil {
			return nil, err
		}
		for _, p := range packets {
			if p.Type != packet.Handshake {
				continue
			}
			var hs HandshakeResponse
			if len(p.Data) > 0 {
				if err := json.Unmarshal(p.Data, &hs); err != nil {
					return nil, err
				}
			}
			if hs.Code != 0 && hs.Code != 200 {
				return nil, fmt.Errorf("handshake failed: code=%d", hs.Code)
			}
			return &hs, nil
		}
	}
}

func (c *Client) readLoop() {
	defer c.Close()

	for {
		if c.readTTL > 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(c.readTTL))
		}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.failPending(err)
			return
		}
		packets, err := c.dec.Decode(data)
		if err != nil {
			c.failPending(err)
			return
		}
		for _, p := range packets {
			c.processPacket(p)
		}
	}
}

func (c *Client) processPacket(p *packet.Packet) {
	switch p.Type {
	case packet.Heartbeat:
		_ = c.writePacket(heartbeatPacket)
	case packet.Data:
		msg, err := message.Decode(p.Data)
		if err != nil {
			return
		}
		if msg.Type == message.Response {
			c.dispatchResponse(msg.ID, response{data: msg.Data})
		}
	case packet.Kick:
		c.failPending(errors.New("kicked by server"))
		c.Close()
	}
}

func (c *Client) failPending(err error) {
	c.mu.Lock()
	responses := c.responses
	c.responses = make(map[uint64]chan response)
	c.mu.Unlock()

	for _, ch := range responses {
		ch <- response{err: err}
	}
}

func buildURL(addr, path string, tls bool) (string, error) {
	if path == "" {
		path = "/nano"
	}
	u := url.URL{
		Scheme: "ws",
		Host:   addr,
		Path:   path,
	}
	if tls {
		u.Scheme = "wss"
	}
	return u.String(), nil
}
