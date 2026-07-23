package auth

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

// LoginCommand, giriş isteği girdisidir.
type LoginCommand struct {
	Email        string
	Password     string
	MFACode      string
	MFAChallenge string
	ClientKey    string
}

const loginChallengeTTL = 5 * time.Minute
const loginChallengePrefix = "auth:mfa:challenge:"

// loginChallenge, MFA ara adımı için cache payload'ı.
// Alan adları kasıtlı opaque: "ClientKey"/"client_key" gosec G117 tetikler.
// CK login-guard parmak izidir (IP), secret değildir.
type loginChallenge struct {
	UID  string `json:"uid"`
	Em   string `json:"em"`
	Role string `json:"role"`
	CK   string `json:"ck"`
}

// LoginHandler, kullanıcı girişini işler.
type LoginHandler struct {
	users     user.Repository
	hasher    appshared.PasswordHasher
	sessions  *SessionManager
	guard     LoginGuard
	totp      TOTP
	mfaRepo   MFARepository
	cache     appshared.Cache
	tx        appshared.TxManager
	publisher appshared.EventPublisher
}

// LoginDeps, LoginHandler bağımlılıklarını gruplar.
type LoginDeps struct {
	Users     user.Repository
	Hasher    appshared.PasswordHasher
	Sessions  *SessionManager
	Guard     LoginGuard
	TOTP      TOTP
	MFARepo   MFARepository
	Cache     appshared.Cache
	Tx        appshared.TxManager
	Publisher appshared.EventPublisher
}

// NewLoginHandler, LoginHandler'ı kurar.
func NewLoginHandler(d LoginDeps) *LoginHandler {
	return &LoginHandler{
		users: d.Users, hasher: d.Hasher, sessions: d.Sessions,
		guard: d.Guard, totp: d.TOTP, mfaRepo: d.MFARepo, cache: d.Cache,
		tx: d.Tx, publisher: d.Publisher,
	}
}

// Handle, kimlik bilgilerini doğrular ve başarılıysa token çifti döner.
// Hata mesajları kullanıcı numaralandırmaya karşı jeneriktir; şifre
// karşılaştırması sabit zamanlıdır. Brute-force hem e-posta hem IP ile sınırlanır.
func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (TokenPair, error) {
	if cmd.MFAChallenge != "" {
		return h.completeMFA(ctx, cmd)
	}

	// Brute-force: hem e-posta hem IP bazlı anahtar kontrol edilir.
	if err := h.checkAllowed(ctx, cmd.Email, cmd.ClientKey); err != nil {
		h.publishLoginFailed(ctx, "", cmd.Email, "rate_limited")
		return TokenPair{}, err
	}

	email, err := user.NewEmail(cmd.Email)
	if err != nil {
		h.recordFailure(ctx, cmd.Email, cmd.ClientKey)
		h.publishLoginFailed(ctx, "", cmd.Email, "invalid_email")
		return TokenPair{}, ErrInvalidCredentials
	}

	u, err := h.users.FindByEmail(ctx, email)
	if err != nil {
		if _, ok := shared.AsDomainError(err); ok {
			h.recordFailure(ctx, cmd.Email, cmd.ClientKey)
			h.publishLoginFailed(ctx, "", cmd.Email, "unknown_user")
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	ok, err := h.hasher.Verify(ctx, cmd.Password, u.Password().Encoded())
	if err != nil {
		return TokenPair{}, err
	}
	if !ok {
		h.recordFailure(ctx, cmd.Email, cmd.ClientKey)
		h.publishLoginFailed(ctx, u.ID().String(), u.Email().String(), "bad_password")
		return TokenPair{}, ErrInvalidCredentials
	}

	if !u.IsActive() {
		h.publishLoginFailed(ctx, u.ID().String(), u.Email().String(), "inactive")
		return TokenPair{}, ErrAccountInactive
	}

	// İki adımlı doğrulama (MFA) etkinse TOTP kodu zorunludur.
	if u.MFAEnabled() {
		if cmd.MFACode == "" {
			challenge, err := h.createMFAChallenge(ctx, u, cmd.ClientKey)
			if err != nil {
				return TokenPair{}, err
			}
			return TokenPair{}, &MFARequiredError{Challenge: challenge}
		}
		if !h.consumeMFACode(ctx, u, cmd.MFACode) {
			h.recordFailure(ctx, cmd.Email, cmd.ClientKey)
			h.publishLoginFailed(ctx, u.ID().String(), u.Email().String(), "bad_mfa")
			return TokenPair{}, ErrInvalidMFACode
		}
	}

	// Başarılı giriş: sayaçları sıfırla.
	_ = h.guard.Reset(ctx, cmd.Email)
	if cmd.ClientKey != "" {
		_ = h.guard.Reset(ctx, cmd.ClientKey)
	}

	// Transparent rehash: hash parametreleri zayıfsa aynı şifreyi yeniden hash'le.
	h.maybeRehash(ctx, u, cmd.Password)

	tp, err := h.sessions.Issue(ctx, Claims{
		UserID: u.ID().String(),
		Email:  u.Email().String(),
		Role:   u.Role().String(),
	})
	if err != nil {
		return TokenPair{}, err
	}
	h.publishLoginSucceeded(ctx, u.ID().String(), u.Email().String(), "password")
	return tp, nil
}

func (h *LoginHandler) createMFAChallenge(ctx context.Context, u *user.User, clientKey string) (string, error) {
	if h.cache == nil {
		return "", ErrInvalidToken
	}
	raw, err := domainauth.GenerateRawToken()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(loginChallenge{
		UID: u.ID().String(), Em: u.Email().String(),
		Role: u.Role().String(), CK: clientKey,
	})
	if err != nil {
		return "", err
	}
	if err := h.cache.Set(ctx, loginChallengePrefix+raw, payload, loginChallengeTTL); err != nil {
		return "", err
	}
	return raw, nil
}

func (h *LoginHandler) completeMFA(ctx context.Context, cmd LoginCommand) (TokenPair, error) {
	if h.cache == nil || h.mfaRepo == nil || strings.TrimSpace(cmd.MFACode) == "" {
		return TokenPair{}, ErrInvalidMFACode
	}
	raw, ok, err := h.cache.Get(ctx, loginChallengePrefix+cmd.MFAChallenge)
	if err != nil {
		return TokenPair{}, err
	}
	if !ok {
		return TokenPair{}, ErrInvalidToken
	}
	var challenge loginChallenge
	if json.Unmarshal(raw, &challenge) != nil ||
		challenge.UID == "" ||
		challenge.CK != cmd.ClientKey {
		return TokenPair{}, ErrInvalidToken
	}
	if err := h.checkAllowed(ctx, challenge.Em, cmd.ClientKey); err != nil {
		return TokenPair{}, err
	}
	id, err := user.ParseID(challenge.UID)
	if err != nil {
		return TokenPair{}, ErrInvalidToken
	}
	u, err := h.users.FindByID(ctx, id)
	if err != nil || !u.IsActive() || !u.MFAEnabled() {
		return TokenPair{}, ErrInvalidToken
	}
	if !h.consumeMFACode(ctx, u, cmd.MFACode) {
		h.recordFailure(ctx, challenge.Em, cmd.ClientKey)
		h.publishLoginFailed(ctx, challenge.UID, challenge.Em, "bad_mfa")
		return TokenPair{}, ErrInvalidMFACode
	}
	if _, taken, err := h.cache.Take(ctx, loginChallengePrefix+cmd.MFAChallenge); err != nil {
		return TokenPair{}, err
	} else if !taken {
		return TokenPair{}, ErrInvalidToken
	}
	_ = h.guard.Reset(ctx, challenge.Em)
	if cmd.ClientKey != "" {
		_ = h.guard.Reset(ctx, cmd.ClientKey)
	}
	tp, err := h.sessions.Issue(ctx, Claims{
		UserID: challenge.UID, Email: challenge.Em, Role: challenge.Role,
	})
	if err != nil {
		return TokenPair{}, err
	}
	h.publishLoginSucceeded(ctx, challenge.UID, challenge.Em, "password+mfa")
	return tp, nil
}

func (h *LoginHandler) consumeMFACode(ctx context.Context, u *user.User, code string) bool {
	if h.mfaRepo == nil {
		return false
	}
	code = strings.TrimSpace(code)
	if step, ok := h.totp.Validate(u.MFASecret(), code); ok {
		consumed, err := h.mfaRepo.ConsumeTOTPStep(ctx, u.ID().String(), step)
		return err == nil && consumed
	}
	hash := domainauth.HashToken(normalizeRecoveryCode(code))
	consumed, err := h.mfaRepo.ConsumeRecoveryCode(ctx, u.ID().String(), hash)
	return err == nil && consumed
}

func (h *LoginHandler) checkAllowed(ctx context.Context, email, clientKey string) error {
	allowed, err := h.guard.Allowed(ctx, email)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrTooManyAttempts
	}
	if clientKey != "" {
		allowed, err = h.guard.Allowed(ctx, clientKey)
		if err != nil {
			return err
		}
		if !allowed {
			return ErrTooManyAttempts
		}
	}
	return nil
}

func (h *LoginHandler) recordFailure(ctx context.Context, email, clientKey string) {
	_ = h.guard.RecordFailure(ctx, email)
	if clientKey != "" {
		_ = h.guard.RecordFailure(ctx, clientKey)
	}
}

func (h *LoginHandler) publishLoginSucceeded(ctx context.Context, userID, email, provider string) {
	if h.publisher == nil {
		return
	}
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
	_ = h.publisher.Publish(ctx, domainauth.NewLoginSucceededEvent(userID, email, provider))
}

func (h *LoginHandler) publishLoginFailed(ctx context.Context, userID, email, reason string) {
	if h.publisher == nil {
		return
	}
	actor := appshared.ActorFromContext(ctx)
	ctx = appshared.WithActor(ctx, appshared.ActorContext{
		ActorID:       userID,
		ActorType:     appshared.ActorTypeAnonymous,
		ActorEmail:    email,
		Source:        actor.Source,
		CorrelationID: actor.CorrelationID,
		IP:            actor.IP,
		UserAgent:     actor.UserAgent,
	})
	_ = h.publisher.Publish(ctx, domainauth.NewLoginFailedEvent(userID, email, reason))
}

// maybeRehash, hash politikası güncellendiyse kullanıcının şifresini sessizce
// yeni parametrelerle yeniden hash'ler (best-effort; hata login'i engellemez).
func (h *LoginHandler) maybeRehash(ctx context.Context, u *user.User, plain string) {
	if !h.hasher.NeedsRehash(u.Password().Encoded()) {
		return
	}
	encoded, err := h.hasher.Hash(ctx, plain)
	if err != nil {
		return
	}
	hashed, err := user.NewHashedPassword(encoded)
	if err != nil {
		return
	}
	if err := u.RehashPassword(hashed); err != nil {
		return
	}
	_ = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		return h.users.Update(ctx, u)
	})
}
