package email

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/uuid"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
)

// AuthNotifier, auth.Notifier portunun outbox tabanlı implementasyonudur.
type AuthNotifier struct {
	dispatch *OutboxDispatcher
	baseURL  string
}

// NewAuthNotifier, notifier'ı kurar.
func NewAuthNotifier(dispatch *OutboxDispatcher, baseURL string) *AuthNotifier {
	return &AuthNotifier{dispatch: dispatch, baseURL: baseURL}
}

// SendEmailVerification, doğrulama bağlantısını içeren e-postayı kuyruğa alır.
func (n *AuthNotifier) SendEmailVerification(ctx context.Context, to, name, token, locale string) error {
	link := n.link("/auth/verify-email", token)
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            to,
		TitleKey:         "email.verify.subject",
		ContentKey:       "email.verify.text",
		HTMLContentKey:   "email.verify.html",
		TitleFallback:    "E-posta adresinizi doğrulayın",
		BodyFallback:     "Merhaba {0},\n\nE-posta adresinizi doğrulamak için bağlantıya tıklayın:\n{1}",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p><a href=\"{1}\">E-posta adresinizi doğrulayın</a></p>",
		Args:             []any{name, link},
		Locale:           locale,
	}, "auth-verify:"+hashKey(to, token))
}

// SendPasswordReset, şifre sıfırlama bağlantısını kuyruğa alır.
func (n *AuthNotifier) SendPasswordReset(ctx context.Context, to, name, token, locale string) error {
	link := n.link("/auth/reset-password", token)
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            to,
		TitleKey:         "email.reset.subject",
		ContentKey:       "email.reset.text",
		HTMLContentKey:   "email.reset.html",
		TitleFallback:    "Şifre sıfırlama",
		BodyFallback:     "Merhaba {0},\n\nŞifrenizi sıfırlamak için bağlantıya tıklayın:\n{1}\n\nBu isteği siz yapmadıysanız yok sayın.",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p><a href=\"{1}\">Şifrenizi sıfırlayın</a></p><p>Bu isteği siz yapmadıysanız yok sayın.</p>",
		Args:             []any{name, link},
		Locale:           locale,
	}, "auth-reset:"+hashKey(to, token))
}

// SendPasswordChanged, şifre değişikliği uyarısını kuyruğa alır.
func (n *AuthNotifier) SendPasswordChanged(ctx context.Context, to, name, locale string) error {
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            to,
		TitleKey:         "email.password_changed.subject",
		ContentKey:       "email.password_changed.text",
		HTMLContentKey:   "email.password_changed.html",
		TitleFallback:    "Şifreniz değiştirildi",
		BodyFallback:     "Merhaba {0},\n\nHesabınızın şifresi değiştirildi. Bu işlemi siz yapmadıysanız derhal bizimle iletişime geçin.",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p>Hesabınızın şifresi değiştirildi. Bu işlemi siz yapmadıysanız derhal bizimle iletişime geçin.</p>",
		Args:             []any{name},
		Locale:           locale,
	}, "") // her değişiklikte yeni iş; idempotency yok
}

func (n *AuthNotifier) link(path, token string) string {
	return n.baseURL + path + "?token=" + url.QueryEscape(token)
}

func hashKey(parts ...string) string {
	return fmt.Sprintf("%x", uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprint(parts))))
}
