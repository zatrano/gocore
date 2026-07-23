package auth

import (
	"context"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
)

// SessionManager, token yaşam döngüsünü yönetir: üretim, doğrulama (iptal
// kontrolü dahil), refresh rotation ve yeniden kullanım (reuse) tespiti.
// TokenIssuer (JWT) ve TokenStore (durum) portlarını birleştirir.
type SessionManager struct {
	issuer    TokenIssuer
	store     TokenStore
	publisher appshared.EventPublisher
}

// NewSessionManager, SessionManager'ı kurar.
func NewSessionManager(issuer TokenIssuer, store TokenStore, publisher appshared.EventPublisher) *SessionManager {
	return &SessionManager{issuer: issuer, store: store, publisher: publisher}
}

// Issue, yeni bir token çifti üretir ve refresh token'ı aktif olarak kaydeder.
func (m *SessionManager) Issue(ctx context.Context, c Claims) (TokenPair, error) {
	tp, err := m.issuer.Issue(ctx, c)
	if err != nil {
		return TokenPair{}, err
	}
	if err := m.registerRefresh(ctx, tp.RefreshToken); err != nil {
		return TokenPair{}, err
	}
	return tp, nil
}

// Verify, access token'ı doğrular ve iptal (logout/toplu iptal) kontrolü yapar.
// Middleware bu metodu kullanır.
func (m *SessionManager) Verify(ctx context.Context, token string) (Claims, error) {
	claims, err := m.issuer.Verify(ctx, token)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	revoked, err := m.store.IsAccessRevoked(ctx, claims.TokenID)
	if err != nil {
		return Claims{}, err
	}
	if revoked {
		return Claims{}, ErrInvalidToken
	}
	// Toplu iptal: token, kullanıcının iptal zaman damgasından önce üretildiyse geçersiz.
	revokedAt, err := m.store.UserRevokedAt(ctx, claims.UserID)
	if err != nil {
		return Claims{}, err
	}
	// Token, toplu iptal anında veya öncesinde üretildiyse geçersizdir.
	if !revokedAt.IsZero() && !claims.IssuedAt.After(revokedAt) {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

// Refresh, geçerli bir refresh token'dan yeni çift üretir (rotation). Tüketilmiş
// bir token yeniden sunulursa reuse tespiti devreye girer ve kullanıcının tüm
// oturumları iptal edilir.
func (m *SessionManager) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	rc, err := m.issuer.Inspect(ctx, refreshToken)
	if err != nil || rc.Type != "refresh" {
		return TokenPair{}, ErrInvalidToken
	}

	active, err := m.store.IsRefreshActive(ctx, rc.UserID, rc.TokenID)
	if err != nil {
		return TokenPair{}, err
	}
	if !active {
		consumed, err := m.store.WasRefreshConsumed(ctx, rc.UserID, rc.TokenID)
		if err != nil {
			return TokenPair{}, err
		}
		if consumed {
			// Rotation sonrası aynı refresh yeniden sunuldu → olası hırsızlık.
			_ = m.store.RevokeAllForUser(ctx, rc.UserID, time.Now().UTC())
			return TokenPair{}, ErrTokenReuse
		}
		// Bilinmeyen / store'da olmayan (ör. restart) veya süresi dolmuş: sessiz geçersiz.
		return TokenPair{}, ErrInvalidToken
	}

	// Rotation: eski token'ı tüket, yenisini üret ve kaydet.
	if err := m.store.ConsumeRefresh(ctx, rc.UserID, rc.TokenID); err != nil {
		return TokenPair{}, err
	}
	tp, err := m.issuer.Issue(ctx, Claims{UserID: rc.UserID, Email: rc.Email, Role: rc.Role})
	if err != nil {
		return TokenPair{}, err
	}
	if err := m.registerRefresh(ctx, tp.RefreshToken); err != nil {
		return TokenPair{}, err
	}
	return tp, nil
}

// Logout, access token'ı iptal eder ve refresh token'ı tüketir.
func (m *SessionManager) Logout(ctx context.Context, accessToken, refreshToken string) error {
	var userID, email string
	if accessToken != "" {
		if ac, err := m.issuer.Inspect(ctx, accessToken); err == nil {
			userID, email = ac.UserID, ac.Email
			_ = m.store.RevokeAccess(ctx, ac.TokenID, ac.ExpiresAt)
		}
	}
	if refreshToken != "" {
		if rc, err := m.issuer.Inspect(ctx, refreshToken); err == nil {
			if userID == "" {
				userID, email = rc.UserID, rc.Email
			}
			_ = m.store.ConsumeRefresh(ctx, rc.UserID, rc.TokenID)
		}
	}
	if m.publisher != nil && userID != "" {
		actor := appshared.ActorFromContext(ctx)
		ctx = appshared.WithActor(ctx, appshared.ActorContext{
			ActorID:       userID,
			ActorType:     appshared.ActorTypeUser,
			ActorEmail:    email,
			Source:        actor.Source,
			CorrelationID: actor.CorrelationID,
			IP:            actor.IP,
			UserAgent:     actor.UserAgent,
		})
		_ = m.publisher.Publish(ctx, domainauth.NewLogoutEvent(userID, email))
	}
	return nil
}

// RevokeAll, bir kullanıcının tüm oturumlarını iptal eder (şifre değişimi vb.).
func (m *SessionManager) RevokeAll(ctx context.Context, userID string) error {
	return m.store.RevokeAllForUser(ctx, userID, time.Now().UTC())
}

func (m *SessionManager) registerRefresh(ctx context.Context, refreshToken string) error {
	rc, err := m.issuer.Inspect(ctx, refreshToken)
	if err != nil {
		return err
	}
	return m.store.ActivateRefresh(ctx, rc.UserID, rc.TokenID, rc.ExpiresAt)
}
