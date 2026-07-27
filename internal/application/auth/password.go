package auth

import (
	"context"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

// ChangePasswordCommand, mevcut şifreyle yeni şifre belirleme girdisidir.
type ChangePasswordCommand struct {
	UserID      string
	OldPassword string
	NewPassword string
}

// ChangePasswordHandler, oturum açmış kullanıcının şifresini değiştirir.
type ChangePasswordHandler struct {
	users    user.Repository
	hasher   appshared.PasswordHasher
	sessions *SessionManager
	notifier Notifier
	pub      appshared.EventPublisher
	tx       appshared.TxManager
}

// ChangePasswordDeps, ChangePasswordHandler bağımlılıklarını gruplar.
type ChangePasswordDeps struct {
	Users    user.Repository
	Hasher   appshared.PasswordHasher
	Sessions *SessionManager
	Notifier Notifier
	Pub      appshared.EventPublisher
	Tx       appshared.TxManager
}

// NewChangePasswordHandler, handler'ı kurar.
func NewChangePasswordHandler(d ChangePasswordDeps) *ChangePasswordHandler {
	return &ChangePasswordHandler{
		users: d.Users, hasher: d.Hasher, sessions: d.Sessions,
		notifier: d.Notifier, pub: d.Pub, tx: d.Tx,
	}
}

// Handle, mevcut şifreyi doğrular, yenisini atar, diğer oturumları iptal eder.
func (h *ChangePasswordHandler) Handle(ctx context.Context, cmd ChangePasswordCommand) error {
	if err := domainauth.ValidatePasswordLength(cmd.NewPassword); err != nil {
		return err
	}
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		ok, err := h.hasher.Verify(ctx, cmd.OldPassword, u.Password().Encoded())
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidCredentials
		}
		if err := applyNewPassword(ctx, h.hasher, h.users, u, cmd.NewPassword); err != nil {
			return err
		}
		return h.pub.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return err
	}

	_ = h.sessions.RevokeAll(ctx, u.ID().String())
	_ = h.notifier.SendPasswordChanged(ctx, u.Email().String(), u.Name(), u.PreferredLocale().String())
	return nil
}

// AdminSetPasswordCommand, yönetici tarafından (eski şifre olmadan) yeni şifre atama girdisidir.
type AdminSetPasswordCommand struct {
	UserID      string
	NewPassword string
}

// AdminSet, eski şifre istemeden yeni şifre atar ve diğer oturumları iptal eder.
func (h *ChangePasswordHandler) AdminSet(ctx context.Context, cmd AdminSetPasswordCommand) error {
	if err := domainauth.ValidatePasswordLength(cmd.NewPassword); err != nil {
		return err
	}
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := applyNewPassword(ctx, h.hasher, h.users, u, cmd.NewPassword); err != nil {
			return err
		}
		return h.pub.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return err
	}

	_ = h.sessions.RevokeAll(ctx, u.ID().String())
	_ = h.notifier.SendPasswordChanged(ctx, u.Email().String(), u.Name(), u.PreferredLocale().String())
	return nil
}

// ForgotPasswordCommand, sıfırlama bağlantısı isteği girdisidir.
type ForgotPasswordCommand struct {
	Email string
}

// ForgotPasswordHandler, sıfırlama token'ı üretir ve e-posta ile gönderir.
type ForgotPasswordHandler struct {
	users    user.Repository
	tokens   domainauth.TokenRepository
	notifier Notifier
	ttl      time.Duration
}

// NewForgotPasswordHandler, handler'ı kurar.
func NewForgotPasswordHandler(users user.Repository, tokens domainauth.TokenRepository, notifier Notifier, ttl time.Duration) *ForgotPasswordHandler {
	return &ForgotPasswordHandler{users: users, tokens: tokens, notifier: notifier, ttl: ttl}
}

// Handle, sıfırlama token'ı üretir (sessiz başarı).
func (h *ForgotPasswordHandler) Handle(ctx context.Context, cmd ForgotPasswordCommand) error {
	email, err := user.NewEmail(cmd.Email)
	if err != nil {
		return nil //nolint:nilerr // geçersiz e-posta: enumeration önleme
	}
	u, err := h.users.FindByEmail(ctx, email)
	if err != nil {
		if _, ok := shared.AsDomainError(err); ok {
			return nil
		}
		return err
	}

	raw, err := domainauth.GenerateRawToken()
	if err != nil {
		return err
	}
	token, err := domainauth.NewOneTimeToken(
		u.ID().String(),
		domainauth.PurposePasswordReset,
		domainauth.HashToken(raw),
		time.Now().UTC().Add(h.ttl),
	)
	if err != nil {
		return err
	}
	_ = h.tokens.DeleteForUser(ctx, u.ID().String(), domainauth.PurposePasswordReset)
	if err := h.tokens.Save(ctx, token); err != nil {
		return err
	}
	return h.notifier.SendPasswordReset(ctx, u.Email().String(), u.Name(), raw, u.PreferredLocale().String())
}

// ResetPasswordCommand, token ile yeni şifre belirleme girdisidir.
type ResetPasswordCommand struct {
	Token       string
	NewPassword string
}

// ResetPasswordHandler, geçerli bir sıfırlama token'ıyla yeni şifre belirler.
type ResetPasswordHandler struct {
	users    user.Repository
	tokens   domainauth.TokenRepository
	hasher   appshared.PasswordHasher
	sessions *SessionManager
	pub      appshared.EventPublisher
	tx       appshared.TxManager
}

// ResetPasswordDeps, ResetPasswordHandler bağımlılıklarını gruplar.
type ResetPasswordDeps struct {
	Users    user.Repository
	Tokens   domainauth.TokenRepository
	Hasher   appshared.PasswordHasher
	Sessions *SessionManager
	Pub      appshared.EventPublisher
	Tx       appshared.TxManager
}

// NewResetPasswordHandler, handler'ı kurar.
func NewResetPasswordHandler(d ResetPasswordDeps) *ResetPasswordHandler {
	return &ResetPasswordHandler{
		users: d.Users, tokens: d.Tokens, hasher: d.Hasher,
		sessions: d.Sessions, pub: d.Pub, tx: d.Tx,
	}
}

// Handle, token'ı doğrular ve yeni şifreyi atar.
func (h *ResetPasswordHandler) Handle(ctx context.Context, cmd ResetPasswordCommand) error {
	if err := domainauth.ValidatePasswordLength(cmd.NewPassword); err != nil {
		return err
	}

	var u *user.User
	err := h.tx.WithinTx(ctx, func(ctx context.Context) error {
		rec, err := h.tokens.FindByHash(ctx, domainauth.HashToken(cmd.Token))
		if err != nil {
			return ErrInvalidToken
		}
		now := time.Now().UTC()
		if err := rec.Consume(domainauth.PurposePasswordReset, now); err != nil {
			return err
		}
		used, err := h.tokens.MarkUsed(ctx, rec.ID())
		if err != nil {
			return err
		}
		if !used {
			return ErrInvalidToken
		}
		id, err := user.ParseID(rec.UserID())
		if err != nil {
			return err
		}
		u, err = h.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := applyNewPassword(ctx, h.hasher, h.users, u, cmd.NewPassword); err != nil {
			return err
		}
		return h.pub.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return err
	}

	_ = h.sessions.RevokeAll(ctx, u.ID().String())
	return nil
}

func applyNewPassword(ctx context.Context, hasher appshared.PasswordHasher, users user.Repository, u *user.User, plain string) error {
	encoded, err := hasher.Hash(ctx, plain)
	if err != nil {
		return err
	}
	hashed, err := user.NewHashedPassword(encoded)
	if err != nil {
		return err
	}
	if err := u.SetPassword(hashed); err != nil {
		return err
	}
	return users.Update(ctx, u)
}
