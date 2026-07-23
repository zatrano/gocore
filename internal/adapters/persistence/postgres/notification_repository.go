package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	"github.com/zatrano/gocore/internal/domain/notification"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/pagination"
)

// NotificationRepository, notification.Repository portunun PostgreSQL
// implementasyonudur.
type NotificationRepository struct {
	tx *database.TxManager
}

// NewNotificationRepository, repository'yi TxManager ile kurar.
func NewNotificationRepository(tx *database.TxManager) *NotificationRepository {
	return &NotificationRepository{tx: tx}
}

func (r *NotificationRepository) queries(ctx context.Context) *db.Queries {
	return db.New(r.tx.DB(ctx))
}

// Save, yeni bir bildirimi kalıcılaştırır.
func (r *NotificationRepository) Save(ctx context.Context, n *notification.Notification) error {
	recipient, err := uuid.Parse(n.RecipientID())
	if err != nil {
		return notification.ErrRecipientRequired
	}
	_, err = r.queries(ctx).CreateNotification(ctx, db.CreateNotificationParams{
		ID:          n.ID().UUID(),
		RecipientID: recipient,
		Title:       n.Title(),
		Content:     n.Content(),
		Read:        n.IsRead(),
		CreatedAt:   ts(n.CreatedAt()),
		ReadAt:      tsPtr(n.ReadAt()),
	})
	return err
}

// ListByRecipient, alıcının bildirimlerini offset sayfalama ile döner.
func (r *NotificationRepository) ListByRecipient(
	ctx context.Context, recipientID string, page pagination.Request, unreadOnly bool,
) (pagination.Page[*notification.Notification], error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return pagination.Page[*notification.Notification]{}, notification.ErrRecipientRequired
	}

	limit := pagination.NormalizeLimit(page.Limit)
	pageNum := page.Page
	if pageNum < 1 {
		pageNum = 1
	}

	var total int64
	if unreadOnly {
		total, err = r.queries(ctx).CountUnreadNotifications(ctx, recipient)
	} else {
		total, err = r.queries(ctx).CountNotificationsByRecipient(ctx, recipient)
	}
	if err != nil {
		return pagination.Page[*notification.Notification]{}, err
	}

	var rows []db.Notification
	if unreadOnly {
		rows, err = r.queries(ctx).ListUnreadNotificationsByRecipientOffset(ctx, db.ListUnreadNotificationsByRecipientOffsetParams{
			RecipientID: recipient,
			Lmt:         pagination.LimitInt32(limit),
			Off:         pagination.OffsetInt32(pageNum, limit),
		})
	} else {
		rows, err = r.queries(ctx).ListNotificationsByRecipientOffset(ctx, db.ListNotificationsByRecipientOffsetParams{
			RecipientID: recipient,
			Lmt:         pagination.LimitInt32(limit),
			Off:         pagination.OffsetInt32(pageNum, limit),
		})
	}
	if err != nil {
		return pagination.Page[*notification.Notification]{}, err
	}

	items := make([]*notification.Notification, 0, len(rows))
	for _, row := range rows {
		items = append(items, notificationToDomain(row))
	}
	return pagination.NewPage(items, pageNum, limit, total), nil
}

// MarkRead, alıcıya ait bildirimi okundu işaretler (sahiplik kontrolü).
func (r *NotificationRepository) MarkRead(ctx context.Context, id notification.ID, recipientID string) error {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return notification.ErrRecipientRequired
	}
	n, err := r.queries(ctx).MarkNotificationRead(ctx, db.MarkNotificationReadParams{
		ID:          id.UUID(),
		RecipientID: recipient,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		// Zaten okunmuş olabilir; kayıt gerçekten var mı diye ayırt et.
		exists, err := r.queries(ctx).NotificationBelongsTo(ctx, db.NotificationBelongsToParams{
			ID:          id.UUID(),
			RecipientID: recipient,
		})
		if err != nil {
			return err
		}
		if !exists {
			return notification.ErrNotFound
		}
	}
	return nil
}

// MarkAllRead, alıcının tüm okunmamış bildirimlerini okundu işaretler.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, recipientID string) (int64, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return 0, notification.ErrRecipientRequired
	}
	return r.queries(ctx).MarkAllNotificationsRead(ctx, recipient)
}

// Delete, alıcıya ait bildirimi siler (sahiplik kontrolü).
func (r *NotificationRepository) Delete(ctx context.Context, id notification.ID, recipientID string) error {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return notification.ErrRecipientRequired
	}
	n, err := r.queries(ctx).DeleteNotification(ctx, db.DeleteNotificationParams{
		ID:          id.UUID(),
		RecipientID: recipient,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return notification.ErrNotFound
	}
	return nil
}

// DeleteAll, alıcının tüm bildirimlerini siler.
func (r *NotificationRepository) DeleteAll(ctx context.Context, recipientID string) (int64, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return 0, notification.ErrRecipientRequired
	}
	return r.queries(ctx).DeleteAllNotificationsByRecipient(ctx, recipient)
}

// CountUnread, alıcının okunmamış bildirim sayısını döner.
func (r *NotificationRepository) CountUnread(ctx context.Context, recipientID string) (int64, error) {
	recipient, err := uuid.Parse(recipientID)
	if err != nil {
		return 0, notification.ErrRecipientRequired
	}
	return r.queries(ctx).CountUnreadNotifications(ctx, recipient)
}

var _ notification.Repository = (*NotificationRepository)(nil)
