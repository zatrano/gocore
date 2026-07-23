package idempotency

import (
	"context"
)

type ctxKey struct{}

// WithKey, context'e idempotency anahtarı ekler (API header veya form nonce).
func WithKey(ctx context.Context, key string) context.Context {
	if key == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, key)
}

// KeyFromContext, context'teki idempotency anahtarını döner.
func KeyFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}
