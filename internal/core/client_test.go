package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/qonhq/qon/internal/config"
)

func TestClientExecuteSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("q") != "1" {
			t.Fatalf("missing query")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	cfg := config.Default()
	cfg.MaxConcurrentRequests = 16
	c := NewClient(cfg)
	defer c.Close()

	resp, err := c.Execute(context.Background(), Request{
		Method: http.MethodGet,
		URL:    ts.URL,
		Query:  map[string]string{"q": "1"},
	}, "")
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status=%d", resp.Status)
	}
	var payload map[string]bool
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !payload["ok"] {
		t.Fatalf("unexpected body")
	}
}

func TestClientRetry(t *testing.T) {
	var calls atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	cfg := config.Default()
	cfg.Retry.MaxAttempts = 3
	cfg.Retry.InitialBackoff = time.Millisecond
	cfg.Retry.MaxBackoff = 5 * time.Millisecond
	c := NewClient(cfg)
	defer c.Close()

	resp, err := c.Execute(context.Background(), Request{Method: http.MethodGet, URL: ts.URL}, "")
	if err != nil {
		t.Fatalf("execute error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("status=%d", resp.Status)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestAccessKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := config.Default()
	cfg.AccessKey = "secret"
	c := NewClient(cfg)
	defer c.Close()

	_, err := c.Execute(context.Background(), Request{Method: http.MethodGet, URL: ts.URL}, "wrong")
	if err == nil {
		t.Fatalf("expected unauthorized error")
	}
}
