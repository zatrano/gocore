package email

import (
	"context"
	"fmt"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
)

// OutboxDispatcher, bildirim komutlarını kalıcı kuyruğa yazar.
type OutboxDispatcher struct {
	enqueuer appoutbox.Enqueuer
}

// NewOutboxDispatcher, dispatcher'ı kurar.
func NewOutboxDispatcher(enqueuer appoutbox.Enqueuer) *OutboxDispatcher {
	return &OutboxDispatcher{enqueuer: enqueuer}
}

// Enqueue, SendCommand'ı notification.dispatch işi olarak kuyruğa ekler.
func (d *OutboxDispatcher) Enqueue(ctx context.Context, cmd appnotif.SendCommand, idempotencyKey string) error {
	args := make([]string, len(cmd.Args))
	for i, a := range cmd.Args {
		args[i] = fmt.Sprint(a)
	}
	payload := appoutbox.DispatchPayload{
		Channel:          string(cmd.Channel),
		UserID:           cmd.UserID,
		Email:            cmd.Email,
		Phone:            cmd.Phone,
		Locale:           cmd.Locale,
		TitleKey:         cmd.TitleKey,
		ContentKey:       cmd.ContentKey,
		HTMLContentKey:   cmd.HTMLContentKey,
		TitleFallback:    cmd.TitleFallback,
		BodyFallback:     cmd.BodyFallback,
		HTMLBodyFallback: cmd.HTMLBodyFallback,
		Args:             args,
	}
	aggID := cmd.UserID
	if aggID == "" {
		aggID = cmd.Email
	}
	if aggID == "" {
		aggID = cmd.Phone
	}
	job, err := appoutbox.NewJob(appoutbox.KindNotificationDispatch, "notification", aggID, idempotencyKey, payload)
	if err != nil {
		return err
	}
	return d.enqueuer.Enqueue(ctx, job)
}
