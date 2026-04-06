package core

import "fmt"

type ErrorKind string

const (
	ErrorNetwork        ErrorKind = "network_error"
	ErrorTimeout        ErrorKind = "timeout_error"
	ErrorTLS            ErrorKind = "tls_error"
	ErrorInvalidRequest ErrorKind = "invalid_request_error"
	ErrorUnauthorized   ErrorKind = "unauthorized_error"
	ErrorRateLimited    ErrorKind = "rate_limited_error"
	ErrorCircuitOpen    ErrorKind = "circuit_open_error"
)

type QonError struct {
	Kind    ErrorKind `json:"kind"`
	Message string    `json:"message"`
	Cause   error     `json:"-"`
}

func (e *QonError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: %s (%v)", e.Kind, e.Message, e.Cause)
}

func classifyError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case contains(msg, "timeout"):
		return &QonError{Kind: ErrorTimeout, Message: "request timed out", Cause: err}
	case contains(msg, "tls") || contains(msg, "certificate"):
		return &QonError{Kind: ErrorTLS, Message: "tls handshake or certificate failure", Cause: err}
	default:
		return &QonError{Kind: ErrorNetwork, Message: "network execution failed", Cause: err}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexFold(haystack, needle) >= 0
}

func indexFold(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if equalFoldASCII(s[i:i+len(substr)], substr) {
			return i
		}
	}
	return -1
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
