package core

import (
	"context"
	"net"
	"sync"
	"time"
)

type dnsEntry struct {
	ip        string
	expiresAt time.Time
}

type DNSCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]dnsEntry
}

func NewDNSCache(ttl time.Duration) *DNSCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &DNSCache{ttl: ttl, entries: make(map[string]dnsEntry)}
}

func (d *DNSCache) Resolve(ctx context.Context, host string) (string, error) {
	now := time.Now()
	d.mu.RLock()
	entry, ok := d.entries[host]
	d.mu.RUnlock()
	if ok && now.Before(entry.expiresAt) {
		return entry.ip, nil
	}

	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", &net.DNSError{Err: "no addresses", Name: host}
	}

	d.mu.Lock()
	d.entries[host] = dnsEntry{ip: addrs[0], expiresAt: now.Add(d.ttl)}
	d.mu.Unlock()
	return addrs[0], nil
}
