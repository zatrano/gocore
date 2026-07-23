package notification

import (
	"context"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	dnotif "github.com/zatrano/gocore/internal/domain/notification"
	"github.com/zatrano/gocore/internal/infrastructure/notification/sms"
)

// EmailChannel, "email" türünü mevcut Mailer üzerinden gönderir.
type EmailChannel struct {
	mailer appshared.Mailer
}

// NewEmailChannel, e-posta kanalını kurar.
func NewEmailChannel(mailer appshared.Mailer) *EmailChannel {
	return &EmailChannel{mailer: mailer}
}

func (c *EmailChannel) Channel() appnotif.Channel { return appnotif.ChannelEmail }

func (c *EmailChannel) Send(ctx context.Context, msg appnotif.RenderedMessage) error {
	return c.mailer.Send(ctx, appshared.Email{
		To:       []string{msg.Email},
		Subject:  msg.Title,
		TextBody: msg.Content,
		HTMLBody: msg.HTMLContent,
	})
}

// SMSChannel, "sms" türünü aktif SMS sağlayıcısı üzerinden gönderir.
type SMSChannel struct {
	provider sms.Provider
}

// NewSMSChannel, SMS kanalını aktif sağlayıcıyla kurar.
func NewSMSChannel(provider sms.Provider) *SMSChannel {
	return &SMSChannel{provider: provider}
}

func (c *SMSChannel) Channel() appnotif.Channel { return appnotif.ChannelSMS }

func (c *SMSChannel) Send(ctx context.Context, msg appnotif.RenderedMessage) error {
	return c.provider.Send(ctx, msg.Phone, msg.Content)
}

// InboxRealtime, in-app kayıt sonrası bağlı canlı istemcileri uyarır
// (genel /api/v1/ws hub; panel, mobil ve API aynı kanalı kullanır).
type InboxRealtime interface {
	NotifyInbox(userID string)
}

// InAppChannel, uygulama içi (inapp) türü kalıcı olarak veritabanına yazar.
type InAppChannel struct {
	repo     dnotif.Repository
	realtime InboxRealtime
}

// NewInAppChannel, uygulama içi bildirim kanalını kurar. realtime nil olabilir.
func NewInAppChannel(repo dnotif.Repository, realtime InboxRealtime) *InAppChannel {
	return &InAppChannel{repo: repo, realtime: realtime}
}

func (c *InAppChannel) Channel() appnotif.Channel { return appnotif.ChannelInApp }

func (c *InAppChannel) Send(ctx context.Context, msg appnotif.RenderedMessage) error {
	n, err := dnotif.New(msg.RecipientUserID, msg.Title, msg.Content)
	if err != nil {
		return err
	}
	if err := c.repo.Save(ctx, n); err != nil {
		return err
	}
	if c.realtime != nil && msg.RecipientUserID != "" {
		c.realtime.NotifyInbox(msg.RecipientUserID)
	}
	return nil
}
