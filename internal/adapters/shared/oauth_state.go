package shared

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
)

const (
	OAuthStateWebPrefix = "web_oauth_state:"
	OAuthStateAPIPrefix = "api_oauth_state:"
	oauthStateTTL       = 10 * time.Minute
)

var ErrOAuthStateUnavailable = errors.New("oauth state önbelleği kullanılamıyor")

// OAuthStatePayload, OAuth başlangıç isteğinin callback'e taşınan güvenli
// bağlamıdır. Provider eşleştirmesi farklı sağlayıcı callback'lerini reddeder.
type OAuthStatePayload struct {
	Provider string `json:"provider"`
	From     string `json:"from,omitempty"`
	Next     string `json:"next,omitempty"`
}

// IssueOAuthState, rastgele state üretip kısa ömürlü olarak saklar.
func IssueOAuthState(
	ctx context.Context,
	cache appshared.Cache,
	prefix string,
	payload OAuthStatePayload,
) (string, error) {
	if cache == nil {
		return "", ErrOAuthStateUnavailable
	}
	state, err := randomOAuthState()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	if err := cache.Set(ctx, prefix+state, raw, oauthStateTTL); err != nil {
		return "", err
	}
	return state, nil
}

// ConsumeOAuthState, state'i atomik olarak tek sefer tüketir.
func ConsumeOAuthState(
	ctx context.Context,
	cache appshared.Cache,
	prefix, state, provider string,
) (OAuthStatePayload, bool) {
	if cache == nil || state == "" || provider == "" {
		return OAuthStatePayload{}, false
	}
	raw, ok, err := cache.Take(ctx, prefix+state)
	if err != nil || !ok {
		return OAuthStatePayload{}, false
	}
	var payload OAuthStatePayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Provider != provider {
		return OAuthStatePayload{}, false
	}
	return payload, true
}

func randomOAuthState() (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
