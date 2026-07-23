package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_disabledSkipsVerify(t *testing.T) {
	t.Parallel()
	c := NewClient("site", "")
	if c.Enabled() {
		t.Fatal("expected disabled client")
	}
	if err := c.Verify(context.Background(), "", "1.2.3.4"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestClient_requiredWhenTokenMissing(t *testing.T) {
	t.Parallel()
	c := NewClient("site", "secret")
	if err := c.Verify(context.Background(), "", "1.2.3.4"); !errors.Is(err, ErrRequired) {
		t.Fatalf("expected ErrRequired, got %v", err)
	}
}

func TestClient_verifySuccess(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	c := NewClient("site", "secret")
	c.verifyURL = srv.URL
	if err := c.Verify(context.Background(), "token", "1.2.3.4"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestClient_verifyFailure(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false})
	}))
	defer srv.Close()

	c := NewClient("site", "secret")
	c.verifyURL = srv.URL
	if err := c.Verify(context.Background(), "bad", "1.2.3.4"); !errors.Is(err, ErrFailed) {
		t.Fatalf("expected ErrFailed, got %v", err)
	}
}
