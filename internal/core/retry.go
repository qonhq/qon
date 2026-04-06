package core

import (
	"errors"
	"net"
	"time"

	"github.com/qonhq/qon/internal/config"
)

func shouldRetry(cfg config.RetryConfig, resp *Response, err error) bool {
	if !cfg.Enabled {
		return false
	}
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) {
			return true
		}
		return true
	}
	if resp == nil {
		return false
	}
	_, ok := cfg.RetryOnStatuses[resp.Status]
	return ok
}

func retryBackoff(cfg config.RetryConfig, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	backoff := float64(cfg.InitialBackoff)
	for i := 1; i < attempt; i++ {
		backoff *= cfg.BackoffMultiplier
	}
	d := time.Duration(backoff)
	if d > cfg.MaxBackoff {
		d = cfg.MaxBackoff
	}
	if d < 0 {
		return 0
	}
	return d
}
