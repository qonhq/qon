package config

import (
	"crypto/tls"
	"time"
)

// Config controls runtime behavior for Qon core.
type Config struct {
	Timeout               time.Duration
	MaxConcurrentRequests int
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	TLSConfig             *tls.Config
	AllowHTTP2            bool
	ProxyURL              string
	Retry                 RetryConfig
	CircuitBreaker        CircuitBreakerConfig
	RateLimit             RateLimitConfig
	LoggingLevel          string
	EnableMetrics         bool
	EnableTracing         bool
	AccessKey             string
}

type RetryConfig struct {
	Enabled           bool
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
	RetryOnStatuses   map[int]struct{}
}

type CircuitBreakerConfig struct {
	Enabled          bool
	FailureThreshold int
	OpenDuration     time.Duration
	HalfOpenRequests int
}

type RateLimitConfig struct {
	Enabled bool
	RPS     int
	Burst   int
}

func Default() Config {
	return Config{
		Timeout:               10 * time.Second,
		MaxConcurrentRequests: 1024,
		MaxIdleConns:          2048,
		MaxIdleConnsPerHost:   256,
		IdleConnTimeout:       90 * time.Second,
		TLSConfig:             &tls.Config{MinVersion: tls.VersionTLS12},
		AllowHTTP2:            true,
		Retry: RetryConfig{
			Enabled:           true,
			MaxAttempts:       3,
			InitialBackoff:    50 * time.Millisecond,
			MaxBackoff:        2 * time.Second,
			BackoffMultiplier: 2,
			RetryOnStatuses: map[int]struct{}{
				408: {},
				429: {},
				500: {},
				502: {},
				503: {},
				504: {},
			},
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 10,
			OpenDuration:     5 * time.Second,
			HalfOpenRequests: 2,
		},
		RateLimit: RateLimitConfig{
			Enabled: false,
			RPS:     1000,
			Burst:   200,
		},
		LoggingLevel:  "info",
		EnableMetrics: true,
		EnableTracing: true,
	}
}
