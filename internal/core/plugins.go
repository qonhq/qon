package core

import "context"

type Plugin interface {
	Name() string
	BeforeRequest(ctx context.Context, req *Request) error
	AfterResponse(ctx context.Context, req *Request, resp *Response) error
	OnError(ctx context.Context, req *Request, err error)
}

type PluginManager struct {
	plugins []Plugin
}

func NewPluginManager(plugins ...Plugin) *PluginManager {
	return &PluginManager{plugins: plugins}
}

func (pm *PluginManager) Add(plugin Plugin) {
	if plugin == nil {
		return
	}
	pm.plugins = append(pm.plugins, plugin)
}

func (pm *PluginManager) BeforeRequest(ctx context.Context, req *Request) error {
	for _, p := range pm.plugins {
		if err := p.BeforeRequest(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PluginManager) AfterResponse(ctx context.Context, req *Request, resp *Response) error {
	for _, p := range pm.plugins {
		if err := p.AfterResponse(ctx, req, resp); err != nil {
			return err
		}
	}
	return nil
}

func (pm *PluginManager) OnError(ctx context.Context, req *Request, err error) {
	for _, p := range pm.plugins {
		p.OnError(ctx, req, err)
	}
}
