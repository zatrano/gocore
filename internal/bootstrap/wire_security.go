package bootstrap

import (
	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/security"
)

func (g *graph) wireSecurity() {
	g.hasher = security.NewArgon2Hasher(security.DefaultArgon2Params())
	issuer := security.NewJWTIssuer(security.JWTConfig{
		Secret:          []byte(g.cfg.Auth.JWTSecret.Value()),
		Issuer:          g.cfg.Auth.JWTIssuer,
		Audience:        g.cfg.Auth.JWTAudience,
		AccessTokenTTL:  g.cfg.Auth.AccessTokenTTL,
		RefreshTokenTTL: g.cfg.Auth.RefreshTokenTTL,
	})
	g.guard = security.NewMemoryLoginGuard(g.cfg.Auth.MaxLoginAttempts, g.cfg.Auth.LockoutDuration)
	g.ipLimiter = security.NewMemoryIPRateLimiter(g.cfg.Security.RateLimitMax, g.cfg.Security.RateLimitWindow)
	g.tokenStore = security.NewMemoryTokenStore()
	g.sessions = appauth.NewSessionManager(issuer, g.tokenStore, g.publisher)
	g.totp = security.NewTOTP(g.cfg.App.Name)
}
