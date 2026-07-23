package auth_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/security"
)

// fakeIssuer, testler için deterministik bir TokenIssuer'dır. Token string'leri
// jti taşır; imza/JWT karmaşasına gerek kalmadan rotation/iptal test edilir.
type fakeIssuer struct {
	mu    sync.Mutex
	seq   int
	store map[string]appauth.Claims
}

func newFakeIssuer() *fakeIssuer {
	return &fakeIssuer{store: make(map[string]appauth.Claims)}
}

func (f *fakeIssuer) Issue(_ context.Context, c appauth.Claims) (appauth.TokenPair, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	aid := fmt.Sprintf("a%d", f.seq)
	f.seq++
	rid := fmt.Sprintf("r%d", f.seq)
	now := time.Now().UTC()
	at := "access-" + aid
	rt := "refresh-" + rid
	f.store[at] = appauth.Claims{UserID: c.UserID, Email: c.Email, Role: c.Role, TokenID: aid, Type: "access", IssuedAt: now, ExpiresAt: now.Add(time.Hour)}
	f.store[rt] = appauth.Claims{UserID: c.UserID, Email: c.Email, Role: c.Role, TokenID: rid, Type: "refresh", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
	return appauth.TokenPair{AccessToken: at, RefreshToken: rt, ExpiresAt: now.Add(time.Hour), TokenType: "Bearer"}, nil
}

func (f *fakeIssuer) Verify(_ context.Context, token string) (appauth.Claims, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.store[token]
	if !ok || c.Type != "access" {
		return appauth.Claims{}, errors.New("invalid")
	}
	return c, nil
}

func (f *fakeIssuer) Inspect(_ context.Context, token string) (appauth.Claims, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.store[token]
	if !ok {
		return appauth.Claims{}, errors.New("invalid")
	}
	return c, nil
}

func newManager() *appauth.SessionManager {
	return appauth.NewSessionManager(newFakeIssuer(), security.NewMemoryTokenStore(), nil)
}

func TestSession_RefreshRotation(t *testing.T) {
	m := newManager()
	ctx := context.Background()

	tp, err := m.Issue(ctx, appauth.Claims{UserID: "u1", Email: "u@x.com", Role: "user"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	// İlk refresh başarılı olmalı ve yeni token üretmeli.
	np, err := m.Refresh(ctx, tp.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if np.RefreshToken == tp.RefreshToken {
		t.Fatal("rotation: refresh token değişmedi")
	}
}

func TestSession_ReuseDetectionRevokesAll(t *testing.T) {
	m := newManager()
	ctx := context.Background()

	tp, _ := m.Issue(ctx, appauth.Claims{UserID: "u1", Role: "user"})

	// Rotation ile eski token tüketilir.
	np, err := m.Refresh(ctx, tp.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// Eski (tüketilmiş) refresh token yeniden kullanılırsa reuse tespiti.
	if _, err := m.Refresh(ctx, tp.RefreshToken); !errors.Is(err, appauth.ErrTokenReuse) {
		t.Fatalf("reuse beklendi, alındı: %v", err)
	}

	// Reuse sonrası, yeni (geçerli görünen) refresh token da geçersiz olmalı.
	if _, err := m.Refresh(ctx, np.RefreshToken); err == nil {
		t.Fatal("reuse sonrası tüm oturumlar iptal edilmeliydi")
	}
}

func TestSession_UnknownRefreshDoesNotRevokeOthers(t *testing.T) {
	issuer := newFakeIssuer()
	store := security.NewMemoryTokenStore()
	m := appauth.NewSessionManager(issuer, store, nil)
	ctx := context.Background()

	orphan, err := m.Issue(ctx, appauth.Claims{UserID: "u1", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}
	live, err := m.Issue(ctx, appauth.Claims{UserID: "u1", Role: "user"})
	if err != nil {
		t.Fatal(err)
	}

	// Restart simülasyonu: yalnızca live refresh store'da kalsın.
	store2 := security.NewMemoryTokenStore()
	m2 := appauth.NewSessionManager(issuer, store2, nil)
	liveClaims, err := issuer.Inspect(ctx, live.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if err := store2.ActivateRefresh(ctx, "u1", liveClaims.TokenID, liveClaims.ExpiresAt); err != nil {
		t.Fatal(err)
	}

	if _, err := m2.Refresh(ctx, orphan.RefreshToken); !errors.Is(err, appauth.ErrInvalidToken) {
		t.Fatalf("bilinmeyen refresh ErrInvalidToken olmalı, got %v", err)
	}
	if _, err := m2.Refresh(ctx, live.RefreshToken); err != nil {
		t.Fatalf("bilinmeyen refresh diğer oturumu iptal etmemeli: %v", err)
	}
}

func TestSession_LogoutRevokesAccess(t *testing.T) {
	m := newManager()
	ctx := context.Background()

	tp, _ := m.Issue(ctx, appauth.Claims{UserID: "u1", Role: "user"})

	if _, err := m.Verify(ctx, tp.AccessToken); err != nil {
		t.Fatalf("access geçerli olmalıydı: %v", err)
	}
	if err := m.Logout(ctx, tp.AccessToken, tp.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := m.Verify(ctx, tp.AccessToken); err == nil {
		t.Fatal("logout sonrası access token iptal edilmeliydi")
	}
	// Tüketilen refresh token artık kullanılamaz.
	if _, err := m.Refresh(ctx, tp.RefreshToken); err == nil {
		t.Fatal("logout sonrası refresh token geçersiz olmalıydı")
	}
}

func TestSession_RevokeAll(t *testing.T) {
	m := newManager()
	ctx := context.Background()

	tp, _ := m.Issue(ctx, appauth.Claims{UserID: "u1", Role: "user"})
	if err := m.RevokeAll(ctx, "u1"); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	if _, err := m.Verify(ctx, tp.AccessToken); err == nil {
		t.Fatal("toplu iptal sonrası access token geçersiz olmalıydı")
	}
}
