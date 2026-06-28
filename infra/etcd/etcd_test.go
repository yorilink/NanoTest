package etcd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestGetUsesGatewayRange(t *testing.T) {
	var gotPath string
	var gotPayload map[string]string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotPath = r.URL.Path
		user, pass, ok := r.BasicAuth()
		if !ok || user != "u" || pass != "p" {
			t.Fatalf("basic auth = %q/%q/%t, want u/p/true", user, pass, ok)
		}
		readPayload(t, r, &gotPayload)
		return jsonResponse(t, http.StatusOK, rangeResponse{Kvs: []keyValue{{Value: encode("bar")}}}), nil
	})

	client := New(Config{
		Endpoints: []string{"http://etcd.local/"},
		Username:  "u",
		Password:  "p",
		Client:    &http.Client{Transport: transport},
	})
	value, ok, err := client.Get(context.Background(), "foo")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if value != "bar" {
		t.Fatalf("value = %q, want %q", value, "bar")
	}
	if gotPath != "/v3/kv/range" {
		t.Fatalf("path = %q, want /v3/kv/range", gotPath)
	}
	if gotPayload["key"] != encode("foo") {
		t.Fatalf("key = %q, want %q", gotPayload["key"], encode("foo"))
	}
}

func TestGetMissingKey(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(t, http.StatusOK, rangeResponse{}), nil
	})

	client := New(Config{
		Endpoints: []string{"http://etcd.local"},
		Client:    &http.Client{Transport: transport},
	})
	value, ok, err := client.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if value != "" {
		t.Fatalf("value = %q, want empty", value)
	}
}

func TestPutAndDeleteUseGatewayPayloads(t *testing.T) {
	seen := make(map[string]map[string]string)
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload map[string]string
		readPayload(t, r, &payload)
		seen[r.URL.Path] = payload
		return jsonResponse(t, http.StatusOK, map[string]string{}), nil
	})

	client := New(Config{
		Endpoints: []string{"http://etcd.local"},
		Client:    &http.Client{Transport: transport},
	})
	if err := client.Put(context.Background(), "room/1", "node-a"); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if err := client.Delete(context.Background(), "room/1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	put := seen["/v3/kv/put"]
	if put["key"] != encode("room/1") || put["value"] != encode("node-a") {
		t.Fatalf("put payload = %#v", put)
	}
	del := seen["/v3/kv/deleterange"]
	if del["key"] != encode("room/1") {
		t.Fatalf("delete payload = %#v", del)
	}
}

func TestDoTriesNextEndpoint(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "bad.local" {
			return stringResponse(http.StatusServiceUnavailable, "unavailable"), nil
		}
		return jsonResponse(t, http.StatusOK, rangeResponse{Kvs: []keyValue{{Value: encode("ok")}}}), nil
	})

	client := New(Config{
		Endpoints: []string{"http://bad.local", "http://good.local"},
		Client:    &http.Client{Transport: transport},
	})
	value, ok, err := client.Get(context.Background(), "key")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok || value != "ok" {
		t.Fatalf("result = %q/%t, want ok/true", value, ok)
	}
}

func TestEncodeDecode(t *testing.T) {
	encoded := encode("hello")
	if encoded != base64.StdEncoding.EncodeToString([]byte("hello")) {
		t.Fatalf("encoded = %q", encoded)
	}
	decoded, err := decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded != "hello" {
		t.Fatalf("decoded = %q, want hello", decoded)
	}
}

func readPayload(t *testing.T, r *http.Request, out interface{}) {
	t.Helper()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body failed: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode body failed: %v", err)
	}
}

func jsonResponse(t *testing.T, status int, value interface{}) *http.Response {
	t.Helper()
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
	return response(status, body.String())
}

func stringResponse(status int, body string) *http.Response {
	return response(status, body)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
