package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/qonhq/qon/internal/core"
)

type Server struct {
	client *core.Client
}

func New(client *core.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/request", s.request)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.client.MetricsSnapshot())
}

func (s *Server) request(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}
	var req struct {
		Method     string            `json:"method"`
		URL        string            `json:"url"`
		Headers    map[string]string `json:"headers"`
		Query      map[string]string `json:"query"`
		BodyBase64 string            `json:"body_base64"`
		TimeoutMS  int64             `json:"timeout_ms"`
		Priority   int               `json:"priority"`
		TraceID    string            `json:"trace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	body, err := base64.StdEncoding.DecodeString(req.BodyBase64)
	if req.BodyBase64 == "" {
		body = nil
		err = nil
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid base64 body"})
		return
	}

	resp, execErr := s.client.Execute(r.Context(), core.Request{
		Method:   req.Method,
		URL:      req.URL,
		Headers:  req.Headers,
		Query:    req.Query,
		Body:     body,
		Timeout:  time.Duration(req.TimeoutMS) * time.Millisecond,
		Priority: req.Priority,
		TraceID:  req.TraceID,
	}, r.Header.Get("X-Qon-Access-Key"))
	if execErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": execErr.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":      resp.Status,
		"headers":     resp.Headers,
		"body_base64": base64.StdEncoding.EncodeToString(resp.Body),
		"duration_ms": resp.Duration.Milliseconds(),
		"trace_id":    resp.TraceID,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
