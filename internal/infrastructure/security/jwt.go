package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/zatrano/gocore/internal/application/auth"
)

// JWTConfig, token üretimi için yapılandırma.
type JWTConfig struct {
	Secret          []byte
	Issuer          string
	Audience        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// JWTIssuer, auth.TokenIssuer portunun HMAC-SHA256 tabanlı implementasyonudur.
// Best practice'ler: kısa ömürlü access token, iss/aud/exp/nbf/iat claim'leri,
// jti (replay koruması için) ve yalnızca HS256 imza algoritmasının kabulü.
type JWTIssuer struct {
	cfg JWTConfig
}

// NewJWTIssuer, JWTIssuer'ı kurar.
func NewJWTIssuer(cfg JWTConfig) *JWTIssuer { return &JWTIssuer{cfg: cfg} }

// customClaims, JWT içine gömülen uygulamaya özel claim'ler.
type customClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	Type  string `json:"typ"` // "access" | "refresh"
	jwt.RegisteredClaims
}

var (
	errInvalidToken   = errors.New("security: geçersiz token")
	errWrongTokenType = errors.New("security: beklenmeyen token tipi")
)

// Issue, verilen claim'lerden access + refresh token çifti üretir.
func (i *JWTIssuer) Issue(_ context.Context, c auth.Claims) (auth.TokenPair, error) {
	now := time.Now().UTC()
	accessExp := now.Add(i.cfg.AccessTokenTTL)

	access, err := i.sign(c, "access", now, accessExp)
	if err != nil {
		return auth.TokenPair{}, err
	}
	refresh, err := i.sign(c, "refresh", now, now.Add(i.cfg.RefreshTokenTTL))
	if err != nil {
		return auth.TokenPair{}, err
	}

	return auth.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    accessExp,
		TokenType:    "Bearer",
	}, nil
}

func (i *JWTIssuer) sign(c auth.Claims, typ string, iat, exp time.Time) (string, error) {
	claims := customClaims{
		Email: c.Email,
		Role:  c.Role,
		Type:  typ,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.cfg.Issuer,
			Subject:   c.UserID,
			Audience:  jwt.ClaimStrings{i.cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(iat),
			IssuedAt:  jwt.NewNumericDate(iat),
			ID:        uuid.NewString(), // jti: replay koruması
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.cfg.Secret)
}

// Verify, access token'ı doğrular ve claim'leri döner.
func (i *JWTIssuer) Verify(_ context.Context, tokenStr string) (auth.Claims, error) {
	claims, err := i.parse(tokenStr, "access")
	if err != nil {
		return auth.Claims{}, err
	}
	return toAuthClaims(claims), nil
}

// Inspect, token'ı tipinden bağımsız olarak çözer (jti/tip/iat/exp için).
// SessionManager rotation ve iptal takibinde kullanır.
func (i *JWTIssuer) Inspect(_ context.Context, tokenStr string) (auth.Claims, error) {
	claims, err := i.parse(tokenStr, "")
	if err != nil {
		return auth.Claims{}, err
	}
	return toAuthClaims(claims), nil
}

// toAuthClaims, JWT claim'lerini application katmanı Claims'ine çevirir.
func toAuthClaims(c *customClaims) auth.Claims {
	out := auth.Claims{
		UserID:  c.Subject,
		Email:   c.Email,
		Role:    c.Role,
		TokenID: c.ID,
		Type:    c.Type,
	}
	if c.IssuedAt != nil {
		out.IssuedAt = c.IssuedAt.Time
	}
	if c.ExpiresAt != nil {
		out.ExpiresAt = c.ExpiresAt.Time
	}
	return out
}

// parse, token'ı doğrular. expectedType boşsa tip kontrolü yapılmaz (Inspect).
func (i *JWTIssuer) parse(tokenStr, expectedType string) (*customClaims, error) {
	claims := &customClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		// Algoritma karıştırma (alg confusion) saldırısına karşı katı kontrol.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: beklenmeyen imza algoritması", errInvalidToken)
		}
		return i.cfg.Secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(i.cfg.Issuer),
		jwt.WithAudience(i.cfg.Audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidToken, err)
	}
	if expectedType != "" && claims.Type != expectedType {
		return nil, errWrongTokenType
	}
	return claims, nil
}

// GenerateSecret, kriptografik olarak güvenli rastgele bir secret üretir
// (config/seed yardımcıları için).
func GenerateSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
