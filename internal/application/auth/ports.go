// Package auth, kimlik doğrulama use-case'lerini içerir: login, token yaşam
// döngüsü (rotation/iptal), şifre değiştirme/sıfırlama, e-posta doğrulama,
// iki adımlı doğrulama (MFA/TOTP) ve OAuth/SSO. Portlar burada tanımlanır,
// implementasyonlar infrastructure katmanındadır (Dependency Inversion).
package auth

import (
	"context"
	"time"
)

// Claims, üretilen/çözümlenen bir token'ın taşıdığı doğrulanmış bilgidir.
type Claims struct {
	UserID string
	Email  string
	Role   string
	// TokenID, JWT jti claim'i (rotation/iptal takibi için).
	TokenID string
	// Type, "access" | "refresh".
	Type string
	// IssuedAt / ExpiresAt, iptal (revocation) pencereleri için.
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// TokenPair, access + refresh token çiftidir.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// TokenIssuer, JWT üretim ve doğrulama portu (infrastructure/security).
type TokenIssuer interface {
	Issue(ctx context.Context, c Claims) (TokenPair, error)
	Verify(ctx context.Context, token string) (Claims, error)
	Inspect(ctx context.Context, token string) (Claims, error)
}

// TokenStore, refresh token rotation, reuse tespiti ve access token iptali portu.
type TokenStore interface {
	ActivateRefresh(ctx context.Context, userID, tokenID string, exp time.Time) error
	IsRefreshActive(ctx context.Context, userID, tokenID string) (bool, error)
	// WasRefreshConsumed, token daha önce rotation/logout ile tüketildi mi?
	// Sunucu restart sonrası store boşken false döner; bilinmeyen token reuse sanılmamalı.
	WasRefreshConsumed(ctx context.Context, userID, tokenID string) (bool, error)
	ConsumeRefresh(ctx context.Context, userID, tokenID string) error
	RevokeAccess(ctx context.Context, tokenID string, exp time.Time) error
	IsAccessRevoked(ctx context.Context, tokenID string) (bool, error)
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
	UserRevokedAt(ctx context.Context, userID string) (time.Time, error)
}

// LoginGuard, brute-force / IP throttling koruması portu.
type LoginGuard interface {
	Allowed(ctx context.Context, key string) (bool, error)
	RecordFailure(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}

// TOTP, iki adımlı doğrulama (RFC 6238) portu.
type TOTP interface {
	Generate(accountName string) (secret string, uri string, err error)
	// Validate, kodun eşleştiği zaman adımını döndürür. Adım persistence
	// katmanında atomik tüketilerek aynı kodun tekrar kullanımı engellenir.
	Validate(secret, code string) (step int64, ok bool)
}

// Notifier, auth ile ilgili e-postaları gönderme portudur.
type Notifier interface {
	SendEmailVerification(ctx context.Context, to, name, token, locale string) error
	SendPasswordReset(ctx context.Context, to, name, token, locale string) error
	SendPasswordChanged(ctx context.Context, to, name, locale string) error
}
