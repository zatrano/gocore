package shared

import (
	"context"
	"testing"

	"github.com/zatrano/gocore/internal/infrastructure/cache"
)

func TestOAuthStateIsSingleUseAndProviderBound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mem := cache.NewMemory()
	want := OAuthStatePayload{
		Provider: "google",
		From:     "register",
		Next:     "/dashboard/account",
	}
	state, err := IssueOAuthState(ctx, mem, OAuthStateWebPrefix, want)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := ConsumeOAuthState(ctx, mem, OAuthStateWebPrefix, state, "google")
	if !ok || got != want {
		t.Fatalf("consume = %#v, %v; want %#v, true", got, ok, want)
	}
	if _, ok := ConsumeOAuthState(ctx, mem, OAuthStateWebPrefix, state, "google"); ok {
		t.Fatal("OAuth state replay accepted")
	}
}

func TestOAuthStateRejectsProviderMismatchAndConsumesState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mem := cache.NewMemory()
	state, err := IssueOAuthState(ctx, mem, OAuthStateWebPrefix, OAuthStatePayload{Provider: "google"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ConsumeOAuthState(ctx, mem, OAuthStateWebPrefix, state, "github"); ok {
		t.Fatal("provider mismatch accepted")
	}
	if _, ok := ConsumeOAuthState(ctx, mem, OAuthStateWebPrefix, state, "google"); ok {
		t.Fatal("mismatched state was not consumed")
	}
}
