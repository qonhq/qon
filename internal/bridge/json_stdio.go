package bridge

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/qonhq/qon/internal/core"
)

type JSONRequest struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	Query      map[string]string `json:"query"`
	BodyBase64 string            `json:"body_base64"`
	TimeoutMS  int64             `json:"timeout_ms"`
	Priority   int               `json:"priority"`
	TraceID    string            `json:"trace_id"`
	AccessKey  string            `json:"access_key"`
}

type JSONResponse struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers"`
	BodyBase64 string            `json:"body_base64,omitempty"`
	DurationMS int64             `json:"duration_ms"`
	TraceID    string            `json:"trace_id,omitempty"`
	Error      *JSONError        `json:"error,omitempty"`
}

type JSONError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

func RunJSONStdio(client *core.Client, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req JSONRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			_ = writeJSON(writer, JSONResponse{Error: &JSONError{Kind: string(core.ErrorInvalidRequest), Message: err.Error()}})
			continue
		}

		body, err := decodeBody(req.BodyBase64)
		if err != nil {
			_ = writeJSON(writer, JSONResponse{Error: &JSONError{Kind: string(core.ErrorInvalidRequest), Message: "invalid body_base64"}})
			continue
		}

		resp, execErr := client.Execute(
			context.Background(),
			core.Request{
				Method:   req.Method,
				URL:      req.URL,
				Headers:  req.Headers,
				Query:    req.Query,
				Body:     body,
				Timeout:  time.Duration(req.TimeoutMS) * time.Millisecond,
				Priority: req.Priority,
				TraceID:  req.TraceID,
			},
			req.AccessKey,
		)
		if execErr != nil {
			kind := string(core.ErrorNetwork)
			if qe, ok := execErr.(*core.QonError); ok {
				kind = string(qe.Kind)
			}
			_ = writeJSON(writer, JSONResponse{Error: &JSONError{Kind: kind, Message: execErr.Error()}})
			continue
		}

		_ = writeJSON(writer, JSONResponse{
			Status:     resp.Status,
			Headers:    resp.Headers,
			BodyBase64: base64.StdEncoding.EncodeToString(resp.Body),
			DurationMS: resp.Duration.Milliseconds(),
			TraceID:    resp.TraceID,
		})
	}
	return scanner.Err()
}

func writeJSON(w *bufio.Writer, v JSONResponse) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	if err := w.WriteByte('\n'); err != nil {
		return err
	}
	return w.Flush()
}

func decodeBody(in string) ([]byte, error) {
	if in == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(in)
}
