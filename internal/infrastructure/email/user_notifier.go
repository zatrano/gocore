package email

import (
	"context"
	"encoding/json"
	"fmt"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	"github.com/zatrano/gocore/internal/domain/user"
)

// UserNotifier, kullanıcı domain event'lerine karşılık bildirim kuyruğuna iş yazar.
type UserNotifier struct {
	dispatch *OutboxDispatcher
	repo     user.Repository
}

// NewUserNotifier, bildirim servisini outbox dispatcher ile kurar.
func NewUserNotifier(dispatch *OutboxDispatcher, repo user.Repository) *UserNotifier {
	return &UserNotifier{dispatch: dispatch, repo: repo}
}

// OnRegisteredPayload, outbox domain event yan etkisi.
func (n *UserNotifier) OnRegisteredPayload(ctx context.Context, p appoutbox.DomainEventPayload) error {
	var ev user.RegisteredEvent
	if err := json.Unmarshal(p.Data, &ev); err != nil {
		return err
	}
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            ev.Email,
		TitleKey:         "email.welcome.subject",
		ContentKey:       "email.welcome.text",
		HTMLContentKey:   "email.welcome.html",
		TitleFallback:    "Hoş geldiniz, {0}!",
		BodyFallback:     "Merhaba {0},\n\n{1} adresiyle hesabınız oluşturuldu.",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p><strong>{1}</strong> adresiyle hesabınız oluşturuldu.</p>",
		Args:             []any{ev.Name, ev.Email},
		Locale:           ev.PreferredLocale,
	}, "user-welcome:"+p.EventID)
}

// OnActivatedPayload, hesap aktifleştirme e-postasını kuyruğa alır.
func (n *UserNotifier) OnActivatedPayload(ctx context.Context, p appoutbox.DomainEventPayload) error {
	u, err := n.loadUser(ctx, p.AggregateID)
	if err != nil {
		return err
	}
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            u.Email().String(),
		TitleKey:         "email.activated.subject",
		ContentKey:       "email.activated.text",
		HTMLContentKey:   "email.activated.html",
		TitleFallback:    "Hesabınız aktifleştirildi",
		BodyFallback:     "Merhaba {0},\n\nHesabınız artık aktif. Giriş yapabilirsiniz.",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p>Hesabınız artık aktif.</p>",
		Args:             []any{u.Name()},
		Locale:           u.PreferredLocale().String(),
	}, "user-activated:"+p.EventID)
}

// OnEmailChangedPayload, e-posta değişikliği bildirimini kuyruğa alır.
func (n *UserNotifier) OnEmailChangedPayload(ctx context.Context, p appoutbox.DomainEventPayload) error {
	var ev user.EmailChangedEvent
	if err := json.Unmarshal(p.Data, &ev); err != nil {
		return err
	}
	u, err := n.loadUser(ctx, p.AggregateID)
	if err != nil {
		return err
	}
	return n.dispatch.Enqueue(ctx, appnotif.SendCommand{
		Channel:          appnotif.ChannelEmail,
		Email:            ev.NewEmail,
		TitleKey:         "email.email_changed.subject",
		ContentKey:       "email.email_changed.text",
		HTMLContentKey:   "email.email_changed.html",
		TitleFallback:    "E-posta adresiniz değiştirildi",
		BodyFallback:     "Merhaba {0},\n\nE-posta adresiniz {1} adresine güncellendi.",
		HTMLBodyFallback: "<p>Merhaba <strong>{0}</strong>,</p><p>E-posta adresiniz <strong>{1}</strong> adresine güncellendi.</p>",
		Args:             []any{u.Name(), ev.NewEmail},
		Locale:           u.PreferredLocale().String(),
	}, "user-email-changed:"+p.EventID)
}

func (n *UserNotifier) loadUser(ctx context.Context, aggregateID string) (*user.User, error) {
	id, err := user.ParseID(aggregateID)
	if err != nil {
		return nil, fmt.Errorf("email: geçersiz kullanıcı kimliği: %w", err)
	}
	return n.repo.FindByID(ctx, id)
}
