-- auth_tokens: tek kullanımlık süreli token'lar (e-posta doğrulama, şifre sıfırlama).
-- Ham token saklanmaz; yalnızca token_hash (SHA-256) tutulur.

-- name: CreateAuthToken :one
INSERT INTO auth_tokens (id, user_id, purpose, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, now())
RETURNING *;

-- name: GetAuthTokenByHash :one
SELECT * FROM auth_tokens WHERE token_hash = $1;

-- name: MarkAuthTokenUsed :execrows
-- Tek kullanım: yalnızca kullanılmamış ve süresi geçmemiş token işaretlenir.
UPDATE auth_tokens
SET used_at = now()
WHERE id = $1 AND used_at IS NULL AND expires_at > now();

-- name: DeleteAuthTokensForUser :exec
-- Aynı amaçlı önceki token'ları geçersiz kılmak için (yeni token üretilirken).
DELETE FROM auth_tokens WHERE user_id = $1 AND purpose = $2;

-- name: DeleteExpiredAuthTokens :exec
-- Periyodik temizlik: süresi dolmuş veya kullanılmış token'ları siler.
DELETE FROM auth_tokens WHERE expires_at < now() OR used_at IS NOT NULL;
