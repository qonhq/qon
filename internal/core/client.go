package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/qonhq/qon/internal/config"
)

type Client struct {
	cfg         config.Config
	httpClient  *http.Client
	metrics     *Metrics
	scheduler   *priorityScheduler
	plugins     *PluginManager
	rateLimiter *tokenBucket
	semaphore   chan struct{}
	dnsCache    *DNSCache
	accessGuard *accessKeyGuard

	breakersMu sync.Mutex
	breakers   map[string]*circuitBreaker

	requestSeq atomic.Uint64
	closed     atomic.Bool
	workersWg  sync.WaitGroup
}

func NewClient(cfg config.Config) *Client {
	transport := &http.Transport{
		Proxy:               proxyFromConfig(cfg.ProxyURL),
		TLSClientConfig:     cfg.TLSConfig,
		ForceAttemptHTTP2:   cfg.AllowHTTP2,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	c := &Client{
		cfg:         cfg,
		httpClient:  &http.Client{Transport: transport},
		metrics:     &Metrics{},
		scheduler:   newPriorityScheduler(),
		plugins:     NewPluginManager(),
		semaphore:   make(chan struct{}, max(1, cfg.MaxConcurrentRequests)),
		dnsCache:    NewDNSCache(30 * time.Second),
		accessGuard: newAccessKeyGuard(cfg.AccessKey),
		breakers:    make(map[string]*circuitBreaker),
	}
	if cfg.RateLimit.Enabled {
		c.rateLimiter = newTokenBucket(cfg.RateLimit.RPS, cfg.RateLimit.Burst)
	}
	workerCount := max(4, min(64, cfg.MaxConcurrentRequests/16+1))
	for i := 0; i < workerCount; i++ {
		c.workersWg.Add(1)
		go c.workerLoop()
	}
	return c
}

func (c *Client) Use(plugin Plugin) {
	c.plugins.Add(plugin)
}

func (c *Client) Execute(ctx context.Context, req Request, accessKey string) (Response, error) {
	if c.closed.Load() {
		return Response{}, &QonError{Kind: ErrorInvalidRequest, Message: "client is closed"}
	}
	if err := c.accessGuard.Validate(ctx, accessKey); err != nil {
		return Response{}, err
	}
	if err := validateRequest(req); err != nil {
		return Response{}, err
	}
	if req.TraceID == "" && c.cfg.EnableTracing {
		req.TraceID = fmt.Sprintf("qon-%d", c.requestSeq.Add(1))
	}

	result := make(chan Result, 1)
	c.scheduler.Submit(RequestTask{Request: req, Result: result})

	select {
	case <-ctx.Done():
		return Response{}, &QonError{Kind: ErrorTimeout, Message: "request cancelled", Cause: ctx.Err()}
	case out := <-result:
		return out.Response, out.Err
	}
}

func (c *Client) MetricsSnapshot() Snapshot {
	return c.metrics.Snapshot()
}

func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.scheduler.Close()
	c.workersWg.Wait()
	if tr, ok := c.httpClient.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

func (c *Client) workerLoop() {
	defer c.workersWg.Done()
	for {
		task, ok := c.scheduler.Next()
		if !ok {
			return
		}
		resp, err := c.executeOne(task.Request)
		task.Result <- Result{Response: resp, Err: err}
		close(task.Result)
	}
}

func (c *Client) executeOne(req Request) (Response, error) {
	ctx := context.Background()
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
		defer cancel()
	}

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return Response{}, &QonError{Kind: ErrorRateLimited, Message: "rate limit wait failed", Cause: err}
		}
	}

	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	c.metrics.RequestsTotal.Add(1)
	c.metrics.InFlight.Add(1)
	defer c.metrics.InFlight.Add(-1)

	if err := c.plugins.BeforeRequest(ctx, &req); err != nil {
		return Response{}, err
	}

	host, err := hostFromURL(req.URL)
	if err != nil {
		return Response{}, err
	}
	breaker := c.breakerForHost(host)
	if c.cfg.CircuitBreaker.Enabled && !breaker.Allow() {
		qerr := &QonError{Kind: ErrorCircuitOpen, Message: "circuit breaker open for host " + host}
		c.plugins.OnError(ctx, &req, qerr)
		c.metrics.RequestsFailed.Add(1)
		return Response{}, qerr
	}

	start := time.Now()
	attempts := max(1, c.cfg.Retry.MaxAttempts)
	var lastErr error
	var resp Response

	for attempt := 1; attempt <= attempts; attempt++ {
		resp, lastErr = c.doHTTP(ctx, req)
		if !shouldRetry(c.cfg.Retry, &resp, lastErr) || attempt == attempts {
			break
		}
		c.metrics.RequestsRetried.Add(1)
		time.Sleep(retryBackoff(c.cfg.Retry, attempt))
	}

	duration := time.Since(start)
	if lastErr != nil {
		if c.cfg.CircuitBreaker.Enabled {
			breaker.OnFailure()
		}
		cerr := classifyError(lastErr)
		c.plugins.OnError(ctx, &req, cerr)
		c.metrics.RequestsFailed.Add(1)
		c.metrics.ObserveLatency(duration)
		return Response{}, cerr
	}

	if c.cfg.CircuitBreaker.Enabled {
		breaker.OnSuccess()
	}
	resp.Duration = duration
	resp.TraceID = req.TraceID
	if err := c.plugins.AfterResponse(ctx, &req, &resp); err != nil {
		c.metrics.RequestsFailed.Add(1)
		return Response{}, err
	}
	c.metrics.RequestsSucceeded.Add(1)
	c.metrics.ObserveLatency(duration)
	return resp, nil
}

func (c *Client) doHTTP(ctx context.Context, req Request) (Response, error) {
	target, err := applyQuery(req.URL, req.Query)
	if err != nil {
		return Response{}, err
	}

	var bodyReader io.Reader
	if req.BodyStream != nil {
		bodyReader = req.BodyStream
	} else {
		bodyReader = bytes.NewReader(req.Body)
		c.metrics.BytesSent.Add(uint64(len(req.Body)))
	}

	hreq, err := http.NewRequestWithContext(ctx, req.Method, target, bodyReader)
	if err != nil {
		return Response{}, &QonError{Kind: ErrorInvalidRequest, Message: "failed building request", Cause: err}
	}
	for k, v := range req.Headers {
		hreq.Header.Set(k, v)
	}
	if req.TraceID != "" {
		hreq.Header.Set("X-Qon-Trace", req.TraceID)
	}

	hresp, err := c.httpClient.Do(hreq)
	if err != nil {
		return Response{}, err
	}

	resp := Response{
		Status:  hresp.StatusCode,
		Headers: flattenHeaders(hresp.Header),
	}
	if req.StreamResponse {
		resp.Stream = hresp.Body
		return resp, nil
	}
	defer hresp.Body.Close()
	payload, err := io.ReadAll(hresp.Body)
	if err != nil {
		return Response{}, err
	}
	c.metrics.BytesReceived.Add(uint64(len(payload)))
	resp.Body = payload
	return resp, nil
}

func (c *Client) breakerForHost(host string) *circuitBreaker {
	c.breakersMu.Lock()
	defer c.breakersMu.Unlock()
	b, ok := c.breakers[host]
	if ok {
		return b
	}
	b = newCircuitBreaker(c.cfg.CircuitBreaker)
	c.breakers[host] = b
	return b
}

func validateRequest(req Request) error {
	if req.Method == "" {
		return &QonError{Kind: ErrorInvalidRequest, Message: "method is required"}
	}
	if req.URL == "" {
		return &QonError{Kind: ErrorInvalidRequest, Message: "url is required"}
	}
	_, err := url.ParseRequestURI(req.URL)
	if err != nil {
		return &QonError{Kind: ErrorInvalidRequest, Message: "invalid url", Cause: err}
	}
	m := strings.ToUpper(req.Method)
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead:
		return nil
	default:
		return &QonError{Kind: ErrorInvalidRequest, Message: "unsupported method"}
	}
}

func applyQuery(base string, query map[string]string) (string, error) {
	if len(query) == 0 {
		return base, nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vals := range h {
		if len(vals) == 0 {
			continue
		}
		out[k] = vals[0]
	}
	return out
}

func hostFromURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Hostname() == "" {
		return "", &QonError{Kind: ErrorInvalidRequest, Message: "url missing host"}
	}
	return u.Hostname(), nil
}

func proxyFromConfig(proxyURL string) func(*http.Request) (*url.URL, error) {
	if strings.TrimSpace(proxyURL) == "" {
		return http.ProxyFromEnvironment
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return http.ProxyFromEnvironment
	}
	return http.ProxyURL(parsed)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
