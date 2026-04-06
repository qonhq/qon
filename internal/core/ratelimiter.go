package core

import (
	"context"
	"time"
)

type tokenBucket struct {
	ch chan struct{}
}

func newTokenBucket(rps, burst int) *tokenBucket {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = rps
	}
	tb := &tokenBucket{ch: make(chan struct{}, burst)}
	for i := 0; i < burst; i++ {
		tb.ch <- struct{}{}
	}
	interval := time.Second / time.Duration(rps)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			select {
			case tb.ch <- struct{}{}:
			default:
			}
		}
	}()
	return tb
}

func (tb *tokenBucket) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-tb.ch:
		return nil
	}
}
