package etcd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Config describes an etcd HTTP gateway client.
type Config struct {
	Endpoints []string
	Username  string
	Password  string
	Timeout   time.Duration
	Client    *http.Client
}

// Client is a lightweight wrapper around the etcd v3 HTTP gateway.
type Client struct {
	endpoints []string
	username  string
	password  string
	client    *http.Client
}

// New creates an etcd HTTP gateway client.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &Client{
		endpoints: normalizeEndpoints(cfg.Endpoints),
		username:  cfg.Username,
		password:  cfg.Password,
		client:    client,
	}
}

// Ping checks whether etcd is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, _, err := c.Get(ctx, "__nano_ping__")
	return err
}

// Get returns a key value and whether it exists.
func (c *Client) Get(ctx context.Context, key string) (string, bool, error) {
	var resp rangeResponse
	if err := c.do(ctx, "/v3/kv/range", map[string]string{
		"key": encode(key),
	}, &resp); err != nil {
		return "", false, err
	}
	if len(resp.Kvs) == 0 {
		return "", false, nil
	}
	value, err := decode(resp.Kvs[0].Value)
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

// Put writes a key value.
func (c *Client) Put(ctx context.Context, key, value string) error {
	return c.do(ctx, "/v3/kv/put", map[string]string{
		"key":   encode(key),
		"value": encode(value),
	}, nil)
}

// Delete deletes a key.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.do(ctx, "/v3/kv/deleterange", map[string]string{
		"key": encode(key),
	}, nil)
}

func (c *Client) do(ctx context.Context, path string, payload interface{}, out interface{}) error {
	if len(c.endpoints) == 0 {
		return fmt.Errorf("etcd endpoints cannot be empty")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for _, endpoint := range c.endpoints {
		req, err := http.NewRequest(http.MethodPost, endpoint+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")
		if c.username != "" || c.password != "" {
			req.SetBasicAuth(c.username, c.password)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		err = decodeResponse(resp, out)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	return lastErr
}

func decodeResponse(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("etcd status %d: %s", resp.StatusCode, string(data))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

func normalizeEndpoints(endpoints []string) []string {
	result := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint = strings.TrimRight(endpoint, "/")
		if endpoint != "" {
			result = append(result, endpoint)
		}
	}
	return result
}

func encode(value string) string {
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func decode(value string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type rangeResponse struct {
	Kvs []keyValue `json:"kvs"`
}

type keyValue struct {
	Value string `json:"value"`
}
