package qon

import (
	"context"

	"github.com/qonhq/qon/internal/config"
	"github.com/qonhq/qon/internal/core"
)

type Client struct {
	inner *core.Client
}

type Request = core.Request
type Response = core.Response
type Snapshot = core.Snapshot

type Config = config.Config

func DefaultConfig() Config {
	return config.Default()
}

func New(cfg Config) *Client {
	return &Client{inner: core.NewClient(cfg)}
}

func (c *Client) Execute(ctx context.Context, req Request, accessKey string) (Response, error) {
	return c.inner.Execute(ctx, req, accessKey)
}

func (c *Client) Metrics() Snapshot {
	return c.inner.MetricsSnapshot()
}

func (c *Client) Close() {
	c.inner.Close()
}
