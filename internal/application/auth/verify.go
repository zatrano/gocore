package auth

import (
	"context"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

// EmailVerifier, e-posta doğrulama token'larını üretir/gönderir ve doğrular.
type EmailVerifier struct {
	users    user.Repository
	tokens   domainauth.TokenRepository
	notifier Notifier
	pub      appshared.EventPublisher
	tx       appshared.TxManager
	ttl      time.Duration
}

// NewEmailVerifier, servisi kurar.
func NewEmailVerifier(
	users user.Repository, tokens domainauth.TokenRepository, notifier Notifier,
	pub appshared.EventPublisher, tx appshared.TxManager, ttl time.Duration,
) *EmailVerifier {
	return &EmailVerifier{users: users, tokens: tokens, notifier: notifier, pub: pub, tx: tx, ttl: ttl}
}

// Send, verilen kullanıcı için yeni bir doğrulama token'ı üretir ve e-posta ile gönderir.
func (v *EmailVerifier) Send(ctx context.Context, u *user.User) error {
	if u.IsEmailVerified() {
		return nil
	}
	raw, err := domainauth.GenerateRawToken()
	if err != nil {
		return err
	}
	token, err := domainauth.NewOneTimeToken(
		u.ID().String(),
		domainauth.PurposeEmailVerify,
		domainauth.HashToken(raw),
		time.Now().UTC().Add(v.ttl),
	)
	if err != nil {
		return err
	}
	_ = v.tokens.DeleteForUser(ctx, u.ID().String(), domainauth.PurposeEmailVerify)
	if err := v.tokens.Save(ctx, token); err != nil {
		return err
	}
	return v.notifier.SendEmailVerification(ctx, u.Email().String(), u.Name(), raw, u.PreferredLocale().String())
}

// ResendCommand, doğrulama e-postasını yeniden gönderme girdisidir.
type ResendCommand struct {
	Email string
}

// Resend, doğrulama e-postasını yeniden gönderir.
func (v *EmailVerifier) Resend(ctx context.Context, cmd ResendCommand) error {
	email, err := user.NewEmail(cmd.Email)
	if err != nil {
		return nil //nolint:nilerr // geçersiz e-posta: enumeration önleme
	}
	u, err := v.users.FindByEmail(ctx, email)
	if err != nil {
		if _, ok := shared.AsDomainError(err); ok {
			return nil
		}
		return err
	}
	return v.Send(ctx, u)
}

// VerifyCommand, e-posta doğrulama token'ı girdisidir.
type VerifyCommand struct {
	Token string
}

// Verify, token'ı doğrular ve kullanıcının e-postasını doğrulanmış işaretler.
func (v *EmailVerifier) Verify(ctx context.Context, cmd VerifyCommand) error {
	var u *user.User
	err := v.tx.WithinTx(ctx, func(ctx context.Context) error {
		rec, err := v.tokens.FindByHash(ctx, domainauth.HashToken(cmd.Token))
		if err != nil {
			return ErrInvalidToken
		}
		now := time.Now().UTC()
		if err := rec.Consume(domainauth.PurposeEmailVerify, now); err != nil {
			return err
		}
		used, err := v.tokens.MarkUsed(ctx, rec.ID())
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
		u, err = v.users.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := u.VerifyEmail(); err != nil {
			return err
		}
		if err := v.users.Update(ctx, u); err != nil {
			return err
		}
		return v.pub.Publish(ctx, u.PullEvents()...)
	})
	return err
}
