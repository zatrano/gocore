-- notifications tablosu için type-safe sorgular (uygulama içi bildirimler).

-- name: CreateNotification :one
INSERT INTO notifications (id, recipient_id, title, content, read, created_at, read_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: MarkNotificationRead :execrows
-- Sahiplik kontrolü: yalnızca alıcıya ait ve okunmamış kayıt güncellenir.
UPDATE notifications
SET read = TRUE,
    read_at = now()
WHERE id = $1 AND recipient_id = $2 AND read = FALSE;

-- name: MarkAllNotificationsRead :execrows
-- Alıcının tüm okunmamış bildirimlerini okundu işaretler.
UPDATE notifications
SET read = TRUE,
    read_at = now()
WHERE recipient_id = $1 AND read = FALSE;

-- name: DeleteNotification :execrows
-- Sahiplik kontrolü: yalnızca alıcıya ait kayıt silinir.
DELETE FROM notifications
WHERE id = $1 AND recipient_id = $2;

-- name: DeleteAllNotificationsByRecipient :execrows
-- Alıcının tüm bildirimlerini siler.
DELETE FROM notifications
WHERE recipient_id = $1;

-- name: NotificationBelongsTo :one
-- Sahiplik doğrulaması (okundu işaretlemede bulunamadı/yetkisiz ayrımı için).
SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1 AND recipient_id = $2) AS exists;

-- name: CountUnreadNotifications :one
SELECT count(*) AS count FROM notifications
WHERE recipient_id = $1 AND read = FALSE;

-- name: CountNotificationsByRecipient :one
SELECT COUNT(*)::bigint AS count
FROM notifications
WHERE recipient_id = sqlc.arg('recipient_id');

-- name: ListNotificationsByRecipientOffset :many
SELECT * FROM notifications
WHERE recipient_id = sqlc.arg('recipient_id')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lmt') OFFSET sqlc.arg('off');

-- name: ListUnreadNotificationsByRecipientOffset :many
SELECT * FROM notifications
WHERE recipient_id = sqlc.arg('recipient_id')
  AND read = FALSE
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lmt') OFFSET sqlc.arg('off');
