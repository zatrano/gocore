-- users tablosu için type-safe sorgular. sqlc bu dosyadan Go kodu üretir.
-- Tüm sorgular parametrelidir ($1, $2 ...) — SQL injection'a karşı birincil savunma.
-- Soft delete: varsayılan okuma sorguları yalnızca deleted_at IS NULL (canlı) kayıtları döner.

-- name: CreateUser :one
INSERT INTO users (id, email, phone, name, password_hash, role, active, email_verified, mfa_enabled, mfa_secret, preferred_locale, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2,
    phone = $3,
    name = $4,
    password_hash = $5,
    role = $6,
    active = $7,
    email_verified = $8,
    mfa_enabled = $9,
    mfa_secret = $10,
    preferred_locale = $11,
    updated_at = $12
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByIDAny :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL;

-- name: ExistsUserByEmail :one
-- Benzersizlik kontrolü yalnızca canlı kayıtlar üzerinden yapılır; böylece
-- soft-delete edilmiş bir e-posta yeni kayıtta yeniden kullanılabilir.
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL) AS exists;

-- name: SoftDeleteUser :execrows
-- Yazılımsal silme: kaydı fiziksel silmek yerine deleted_at damgalar.
-- Zaten silinmiş kayıtlar etkilenmez (idempotent; execrows=0 → bulunamadı).
UPDATE users
SET deleted_at = now(),
    updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: RestoreUser :execrows
-- Yazılımsal silmeyi geri alır (deleted_at = NULL).
UPDATE users
SET deleted_at = NULL,
    updated_at = now()
WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: HardDeleteUser :execrows
-- Kalıcı (fiziksel) silme. Yalnızca zaten soft-delete edilmiş kayıtlar için;
-- kaza sonucu canlı kaydın kaybını önler (ör. GDPR "kalıcı silme" akışı).
DELETE FROM users WHERE id = $1 AND deleted_at IS NOT NULL;

-- name: GetUsersByIDs :many
-- Batch okuma: N+1 sorgusunu önlemek için tek sorguda birden çok kullanıcı.
-- ANY($1) ile UUID dizisi tek parametre olarak geçirilir.
SELECT * FROM users WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL;

-- name: CountUsers :one
SELECT COUNT(*)::bigint AS count
FROM users
WHERE (
        (sqlc.narg('deleted')::text IS NULL AND deleted_at IS NULL)
        OR (sqlc.narg('deleted')::text = 'only' AND deleted_at IS NOT NULL)
        OR (sqlc.narg('deleted')::text = 'all')
      )
  AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
  AND (
        sqlc.narg('search')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('search') || '%'
        OR email ILIKE '%' || sqlc.narg('search') || '%'
      );

-- name: ListUsersDescOffset :many
SELECT * FROM users
WHERE (
        (sqlc.narg('deleted')::text IS NULL AND deleted_at IS NULL)
        OR (sqlc.narg('deleted')::text = 'only' AND deleted_at IS NOT NULL)
        OR (sqlc.narg('deleted')::text = 'all')
      )
  AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
  AND (
        sqlc.narg('search')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('search') || '%'
        OR email ILIKE '%' || sqlc.narg('search') || '%'
      )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lmt') OFFSET sqlc.arg('off');

-- name: ListUsersAscOffset :many
SELECT * FROM users
WHERE (
        (sqlc.narg('deleted')::text IS NULL AND deleted_at IS NULL)
        OR (sqlc.narg('deleted')::text = 'only' AND deleted_at IS NOT NULL)
        OR (sqlc.narg('deleted')::text = 'all')
      )
  AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
  AND (
        sqlc.narg('search')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('search') || '%'
        OR email ILIKE '%' || sqlc.narg('search') || '%'
      )
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg('lmt') OFFSET sqlc.arg('off');

-- name: ListUsersDescKeyset :many
SELECT * FROM users
WHERE (
        (sqlc.narg('deleted')::text IS NULL AND deleted_at IS NULL)
        OR (sqlc.narg('deleted')::text = 'only' AND deleted_at IS NOT NULL)
        OR (sqlc.narg('deleted')::text = 'all')
      )
  AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
  AND (
        sqlc.narg('search')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('search') || '%'
        OR email ILIKE '%' || sqlc.narg('search') || '%'
      )
  AND (created_at, id) < (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_id')::uuid)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lmt');

-- name: ListUsersAscKeyset :many
SELECT * FROM users
WHERE (
        (sqlc.narg('deleted')::text IS NULL AND deleted_at IS NULL)
        OR (sqlc.narg('deleted')::text = 'only' AND deleted_at IS NOT NULL)
        OR (sqlc.narg('deleted')::text = 'all')
      )
  AND (sqlc.narg('role')::text IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
  AND (
        sqlc.narg('search')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('search') || '%'
        OR email ILIKE '%' || sqlc.narg('search') || '%'
      )
  AND (created_at, id) > (sqlc.arg('cursor_created_at'), sqlc.arg('cursor_id')::uuid)
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg('lmt');

-- name: CountActiveUsersByRole :one
SELECT COUNT(*)::bigint AS count
FROM users
WHERE role = $1 AND deleted_at IS NULL;
