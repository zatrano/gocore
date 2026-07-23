package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	"github.com/zatrano/gocore/internal/domain/notification"
)

// notificationToDomain, sqlc satırını domain aggregate'ine (yeniden) oluşturur.
func notificationToDomain(row db.Notification) *notification.Notification {
	var readAt *time.Time
	if row.ReadAt.Valid {
		t := row.ReadAt.Time
		readAt = &t
	}
	return notification.Hydrate(
		notification.IDFromUUID(row.ID),
		row.RecipientID.String(),
		row.Title,
		row.Content,
		row.Read,
		row.CreatedAt.Time,
		readAt,
	)
}

// tsPtr, opsiyonel time.Time'ı pgtype.Timestamptz'a çevirir (nil → NULL).
func tsPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
