package core

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	RequestsTotal     atomic.Uint64
	RequestsSucceeded atomic.Uint64
	RequestsFailed    atomic.Uint64
	RequestsRetried   atomic.Uint64
	BytesSent         atomic.Uint64
	BytesReceived     atomic.Uint64
	InFlight          atomic.Int64
	TotalLatencyNanos atomic.Uint64
	LatencySamples    atomic.Uint64
}

type Snapshot struct {
	RequestsTotal     uint64  `json:"requests_total"`
	RequestsSucceeded uint64  `json:"requests_succeeded"`
	RequestsFailed    uint64  `json:"requests_failed"`
	RequestsRetried   uint64  `json:"requests_retried"`
	BytesSent         uint64  `json:"bytes_sent"`
	BytesReceived     uint64  `json:"bytes_received"`
	InFlight          int64   `json:"in_flight"`
	AverageLatencyMS  float64 `json:"average_latency_ms"`
}

func (m *Metrics) ObserveLatency(d time.Duration) {
	m.TotalLatencyNanos.Add(uint64(d.Nanoseconds()))
	m.LatencySamples.Add(1)
}

func (m *Metrics) Snapshot() Snapshot {
	totalNanos := m.TotalLatencyNanos.Load()
	samples := m.LatencySamples.Load()
	avg := 0.0
	if samples > 0 {
		avg = float64(totalNanos) / float64(samples) / float64(time.Millisecond)
	}
	return Snapshot{
		RequestsTotal:     m.RequestsTotal.Load(),
		RequestsSucceeded: m.RequestsSucceeded.Load(),
		RequestsFailed:    m.RequestsFailed.Load(),
		RequestsRetried:   m.RequestsRetried.Load(),
		BytesSent:         m.BytesSent.Load(),
		BytesReceived:     m.BytesReceived.Load(),
		InFlight:          m.InFlight.Load(),
		AverageLatencyMS:  avg,
	}
}
