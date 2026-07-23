package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

const idempotencyUniqueViolation = "23505"

// IdempotencyRepository, idempotency_keys tablosu erişimi.
type IdempotencyRepository struct {
	tx *database.TxManager
}

// NewIdempotencyRepository, repository'yi kurar.
func NewIdempotencyRepository(tx *database.TxManager) *IdempotencyRepository {
	return &IdempotencyRepository{tx: tx}
}

// Insert, yeni idempotency kaydı ekler.
func (r *IdempotencyRepository) Insert(ctx context.Context, rec appidempotency.Record) error {
	id, err := uuid.Parse(rec.ID)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `
		INSERT INTO idempotency_keys (id, scope, key, actor_id, request_hash, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, rec.Scope, rec.Key, rec.ActorID, rec.RequestHash, string(rec.Status), rec.ExpiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == idempotencyUniqueViolation {
			return appidempotency.ErrInProgress
		}
	}
	return err
}

// Find, scope+key+actor ile kayıt getirir.
func (r *IdempotencyRepository) Find(ctx context.Context, scope, key, actorID string) (appidempotency.Record, error) {
	row := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT id, scope, key, actor_id, request_hash, status, response, expires_at
		FROM idempotency_keys
		WHERE scope = $1 AND key = $2 AND actor_id = $3 AND expires_at > now()
	`, scope, key, actorID)
	var (
		id        uuid.UUID
		reqHash   string
		status    string
		response  []byte
		expiresAt time.Time
		out       appidempotency.Record
	)
	err := row.Scan(&id, &out.Scope, &out.Key, &out.ActorID, &reqHash, &status, &response, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appidempotency.Record{}, pgx.ErrNoRows
		}
		return appidempotency.Record{}, err
	}
	out.ID = id.String()
	out.RequestHash = reqHash
	out.Status = appidempotency.Status(status)
	out.Response = response
	out.ExpiresAt = expiresAt
	return out, nil
}

// Complete, kaydı tamamlanmış olarak işaretler.
func (r *IdempotencyRepository) Complete(ctx context.Context, id string, response []byte) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `
		UPDATE idempotency_keys SET status = 'completed', response = $2 WHERE id = $1
	`, uid, response)
	return err
}

// Fail, kaydı başarısız olarak işaretler.
func (r *IdempotencyRepository) Fail(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `
		UPDATE idempotency_keys SET status = 'failed' WHERE id = $1
	`, uid)
	return err
}

var _ appidempotency.Repository = (*IdempotencyRepository)(nil)
