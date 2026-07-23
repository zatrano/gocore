package idempotency_test

import (
	"context"
	"testing"

	"github.com/zatrano/gocore/internal/application/idempotency"
)

func TestRunHTTP_IdempotentReplay(t *testing.T) {
	repo := &memRepo{}
	svc := idempotency.NewService(repo, 0)
	calls := 0
	fn := func() (*idempotency.HTTPStoredResponse, error) {
		calls++
		return &idempotency.HTTPStoredResponse{StatusCode: 202, Body: []byte(`{"ok":true}`)}, nil
	}
	_, _, err := svc.RunHTTP(context.Background(), idempotency.ScopeAPI+":POST:/api/v1/notifications/send", "k1", "actor", "hash", fn)
	if err != nil {
		t.Fatal(err)
	}
	cached, resp, err := svc.RunHTTP(context.Background(), idempotency.ScopeAPI+":POST:/api/v1/notifications/send", "k1", "actor", "hash", fn)
	if err != nil {
		t.Fatal(err)
	}
	if !cached {
		t.Fatal("expected cached replay")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if resp.StatusCode != 202 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestRunHTTP_NoKeyAlwaysRuns(t *testing.T) {
	repo := &memRepo{}
	svc := idempotency.NewService(repo, 0)
	calls := 0
	fn := func() (*idempotency.HTTPStoredResponse, error) {
		calls++
		return &idempotency.HTTPStoredResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
	}
	for i := 0; i < 2; i++ {
		if _, _, err := svc.RunHTTP(context.Background(), "s", "", "a", "", fn); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

// memRepo from service_test.go - same package
