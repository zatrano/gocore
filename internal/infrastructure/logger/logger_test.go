package logger_test

import (
	"context"
	"testing"

	"github.com/zatrano/gocore/internal/infrastructure/logger"
)

func TestWithRequestClientPreservesFields(t *testing.T) {
	ctx := context.Background()
	ctx = logger.WithCorrelationID(ctx, "corr-1")
	ctx = logger.WithRequestClient(ctx, "127.0.0.1", "Mozilla/5.0 Test")
	ctx = logger.WithUserID(ctx, "user-1")

	if got := logger.CorrelationID(ctx); got != "corr-1" {
		t.Fatalf("correlation id = %q, want corr-1", got)
	}
	if got := logger.UserID(ctx); got != "user-1" {
		t.Fatalf("user id = %q, want user-1", got)
	}
	if got := logger.ClientIP(ctx); got != "127.0.0.1" {
		t.Fatalf("client ip = %q, want 127.0.0.1", got)
	}
	if got := logger.UserAgent(ctx); got != "Mozilla/5.0 Test" {
		t.Fatalf("user agent = %q, want Mozilla/5.0 Test", got)
	}
}
