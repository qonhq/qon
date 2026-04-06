package core

import "context"

type accessKeyGuard struct {
	required string
}

func newAccessKeyGuard(required string) *accessKeyGuard {
	return &accessKeyGuard{required: required}
}

func (a *accessKeyGuard) Validate(ctx context.Context, provided string) error {
	_ = ctx
	if a.required == "" {
		return nil
	}
	if provided != a.required {
		return &QonError{Kind: ErrorUnauthorized, Message: "invalid access key"}
	}
	return nil
}
