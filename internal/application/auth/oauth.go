package auth

import (
	"context"
	"sort"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

// OAuthUserInfo, sağlayıcıdan dönen normalize edilmiş kullanıcı bilgisidir.
type OAuthUserInfo struct {
	Email          string
	Name           string
	ProviderUserID string
}

// OAuthProvider, tek bir OAuth/OIDC sağlayıcısını (google, github, ...) temsil
// eden porttur. İmplementasyonlar infrastructure katmanındadır.
type OAuthProvider interface {
	// Name, sağlayıcının adını döner (ör. "google").
	Name() string
	// AuthCodeURL, kullanıcıyı yönlendireceğimiz yetkilendirme URL'ini üretir.
	AuthCodeURL(state string) string
	// Exchange, callback'te dönen kodu kullanıcı bilgisine çevirir.
	Exchange(ctx context.Context, code string) (OAuthUserInfo, error)
}

// OAuthHandler, OAuth/SSO ile giriş akışını yönetir: yönlendirme URL'i üretimi
// ve callback'te kullanıcı bul-veya-oluştur + token üretimi.
type OAuthHandler struct {
	providers     map[string]OAuthProvider
	users         user.Repository
	hasher        appshared.PasswordHasher
	sessions      *SessionManager
	pub           appshared.EventPublisher
	tx            appshared.TxManager
	defaultLocale string
}

// OAuthDeps, OAuthHandler bağımlılıklarını gruplar.
type OAuthDeps struct {
	Providers     []OAuthProvider
	Users         user.Repository
	Hasher        appshared.PasswordHasher
	Sessions      *SessionManager
	Pub           appshared.EventPublisher
	Tx            appshared.TxManager
	DefaultLocale string
}

// NewOAuthHandler, handler'ı kayıtlı sağlayıcılarla kurar.
func NewOAuthHandler(d OAuthDeps) *OAuthHandler {
	m := make(map[string]OAuthProvider, len(d.Providers))
	for _, p := range d.Providers {
		m[p.Name()] = p
	}
	return &OAuthHandler{
		providers: m, users: d.Users, hasher: d.Hasher, sessions: d.Sessions,
		pub: d.Pub, tx: d.Tx, defaultLocale: d.DefaultLocale,
	}
}

// Providers, yapılandırılmış OAuth sağlayıcı adlarını döner.
func (h *OAuthHandler) Providers() []string {
	names := make([]string, 0, len(h.providers))
	for name := range h.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AuthCodeURL, verilen sağlayıcı için yönlendirme URL'ini döner.
func (h *OAuthHandler) AuthCodeURL(provider, state string) (string, error) {
	p, ok := h.providers[provider]
	if !ok {
		return "", ErrOAuthProviderNotFound
	}
	return p.AuthCodeURL(state), nil
}

// Callback, sağlayıcıdan dönen kodu işler: kullanıcıyı bulur veya oluşturur ve
// token çifti üretir. OAuth ile gelen e-posta doğrulanmış kabul edilir.
func (h *OAuthHandler) Callback(ctx context.Context, provider, code string) (TokenPair, error) {
	p, ok := h.providers[provider]
	if !ok {
		return TokenPair{}, ErrOAuthProviderNotFound
	}
	info, err := p.Exchange(ctx, code)
	if err != nil {
		return TokenPair{}, ErrOAuthExchange
	}
	email, err := user.NewEmail(info.Email)
	if err != nil {
		return TokenPair{}, ErrOAuthExchange
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		existing, ferr := h.users.FindByEmail(ctx, email)
		if ferr == nil {
			u = existing
			return h.pub.Publish(ctx, u.PullEvents()...)
		}
		if _, ok := shared.AsDomainError(ferr); !ok {
			return ferr
		}
		// Kullanıcı yok → OAuth ile oluştur (aktif + e-posta doğrulanmış).
		created, cerr := h.provisionUser(ctx, email, info.Name)
		if cerr != nil {
			return cerr
		}
		u = created
		return h.pub.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return TokenPair{}, err
	}

	if !u.IsActive() {
		return TokenPair{}, ErrAccountInactive
	}
	tp, err := h.sessions.Issue(ctx, Claims{
		UserID: u.ID().String(),
		Email:  u.Email().String(),
		Role:   u.Role().String(),
	})
	if err != nil {
		return TokenPair{}, err
	}
	if h.pub != nil {
		actor := appshared.ActorFromContext(ctx)
		pubCtx := appshared.WithActor(ctx, appshared.ActorContext{
			ActorID:       u.ID().String(),
			ActorType:     appshared.ActorTypeUser,
			ActorEmail:    u.Email().String(),
			Source:        actor.Source,
			CorrelationID: actor.CorrelationID,
			IP:            actor.IP,
			UserAgent:     actor.UserAgent,
		})
		_ = h.pub.Publish(pubCtx, domainauth.NewLoginSucceededEvent(u.ID().String(), u.Email().String(), provider))
	}
	return tp, nil
}

// provisionUser, OAuth ile gelen yeni kullanıcıyı oluşturur: rastgele şifre,
// aktif hesap ve doğrulanmış e-posta.
func (h *OAuthHandler) provisionUser(ctx context.Context, email user.Email, name string) (*user.User, error) {
	random, err := domainauth.GenerateRawToken()
	if err != nil {
		return nil, err
	}
	encoded, err := h.hasher.Hash(ctx, random)
	if err != nil {
		return nil, err
	}
	hashed, err := user.NewHashedPassword(encoded)
	if err != nil {
		return nil, err
	}
	locale, err := user.ParsePreferredLocale(h.defaultLocale)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = email.String()
	}
	u, err := user.Register(email, name, hashed, user.RoleUser, locale, user.Phone{})
	if err != nil {
		return nil, err
	}
	if err := u.Activate(); err != nil {
		return nil, err
	}
	if err := u.VerifyEmail(); err != nil {
		return nil, err
	}
	if err := h.users.Save(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
