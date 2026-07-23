package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRetryable_HTTP5xx(t *testing.T) {
	err := MarkRetryable(fmt.Errorf("iyzico: http 503: unavailable"))
	if !Retryable(err) {
		t.Fatal("expected retryable")
	}
}

func TestRetryable_BusinessError(t *testing.T) {
	err := fmt.Errorf("iyzico: card declined (10005)")
	if Retryable(err) {
		t.Fatal("expected non-retryable")
	}
}

func TestDo_SucceedsAfterTransient(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), Options{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}, func() error {
		attempts++
		if attempts < 2 {
			return MarkRetryable(errors.New("timeout"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestDo_StopsOnPermanentError(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), Options{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}, func() error {
		attempts++
		return fmt.Errorf("validation failed")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}
