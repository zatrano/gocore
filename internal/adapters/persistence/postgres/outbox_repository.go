package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

// OutboxRepository, appoutbox.Repository PostgreSQL implementasyonudur.
type OutboxRepository struct {
	tx *database.TxManager
}

// NewOutboxRepository, repository'yi kurar.
func NewOutboxRepository(tx *database.TxManager) *OutboxRepository {
	return &OutboxRepository{tx: tx}
}

// Enqueue, işi kuyruğa ekler. Aynı (kind, idempotency_key) çifti varsa no-op.
func (r *OutboxRepository) Enqueue(ctx context.Context, job appoutbox.Job) error {
	id := uuid.New()
	if job.ID != "" {
		parsed, err := uuid.Parse(job.ID)
		if err != nil {
			return err
		}
		id = parsed
	}
	maxAttempts := job.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	availableAt := job.AvailableAt
	if availableAt.IsZero() {
		availableAt = time.Now().UTC()
	}
	payload := job.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	var idem any
	if job.IdempotencyKey != "" {
		idem = job.IdempotencyKey
	}

	_, err := r.tx.DB(ctx).Exec(ctx, `
		INSERT INTO outbox_jobs (
			id, kind, aggregate_type, aggregate_id, idempotency_key, payload,
			status, attempts, max_attempts, available_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,'pending',0,$7,$8,now())
		ON CONFLICT DO NOTHING
	`, id, job.Kind, job.AggregateType, job.AggregateID, idem, payload, maxAttempts, availableAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	return nil
}

// Claim, bekleyen veya süresi dolmuş işleri lease ile alır.
func (r *OutboxRepository) Claim(ctx context.Context, limit int, lease time.Duration) ([]appoutbox.Job, error) {
	if limit <= 0 {
		limit = 10
	}
	if lease <= 0 {
		lease = 30 * time.Second
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(lease)

	rows, err := r.tx.DB(ctx).Query(ctx, `
		WITH candidates AS (
			SELECT id
			FROM outbox_jobs
			WHERE (
				status IN ('pending', 'failed') AND available_at <= $1
			) OR (
				status = 'processing' AND lease_until IS NOT NULL AND lease_until < $1
			)
			ORDER BY available_at ASC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE outbox_jobs j
		SET status = 'processing',
		    lease_until = $3,
		    attempts = j.attempts + 1
		FROM candidates c
		WHERE j.id = c.id
		RETURNING j.id, j.kind, j.aggregate_type, j.aggregate_id, COALESCE(j.idempotency_key, ''),
		          j.payload, j.status, j.attempts, j.max_attempts, j.available_at,
		          j.lease_until, COALESCE(j.last_error, ''), j.created_at, j.completed_at
	`, now, limit, leaseUntil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []appoutbox.Job
	for rows.Next() {
		var (
			id                     uuid.UUID
			job                    appoutbox.Job
			leasePtr, completedPtr *time.Time
			availableAt, createdAt time.Time
		)
		if err := rows.Scan(
			&id, &job.Kind, &job.AggregateType, &job.AggregateID, &job.IdempotencyKey,
			&job.Payload, &job.Status, &job.Attempts, &job.MaxAttempts, &availableAt,
			&leasePtr, &job.LastError, &createdAt, &completedPtr,
		); err != nil {
			return nil, err
		}
		job.ID = id.String()
		job.AvailableAt = availableAt
		job.LeaseUntil = leasePtr
		job.CreatedAt = createdAt
		job.CompletedAt = completedPtr
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// MarkCompleted, işi tamamlandı olarak işaretler.
func (r *OutboxRepository) MarkCompleted(ctx context.Context, id string) error {
	_, err := r.tx.DB(ctx).Exec(ctx, `
		UPDATE outbox_jobs
		SET status = 'completed', lease_until = NULL, completed_at = now(), last_error = NULL
		WHERE id = $1
	`, uuid.MustParse(id))
	return err
}

// MarkRetryable, işi yeniden denenebilir hale getirir.
func (r *OutboxRepository) MarkRetryable(ctx context.Context, id string, attempts int, nextAttempt time.Time, lastErr string) error {
	_, err := r.tx.DB(ctx).Exec(ctx, `
		UPDATE outbox_jobs
		SET status = 'failed', lease_until = NULL, attempts = $2,
		    available_at = $3, last_error = $4
		WHERE id = $1
	`, uuid.MustParse(id), attempts, nextAttempt, truncateErr(lastErr))
	return err
}

// MarkDead, işi dead-letter olarak işaretler.
func (r *OutboxRepository) MarkDead(ctx context.Context, id string, attempts int, lastErr string) error {
	_, err := r.tx.DB(ctx).Exec(ctx, `
		UPDATE outbox_jobs
		SET status = 'dead', lease_until = NULL, attempts = $2,
		    last_error = $3, completed_at = now()
		WHERE id = $1
	`, uuid.MustParse(id), attempts, truncateErr(lastErr))
	return err
}

// Stats, kuyruk durum özetini döner.
func (r *OutboxRepository) Stats(ctx context.Context) (appoutbox.Stats, error) {
	var s appoutbox.Stats
	err := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending')::bigint,
			COUNT(*) FILTER (WHERE status = 'processing')::bigint,
			COUNT(*) FILTER (WHERE status = 'failed')::bigint,
			COUNT(*) FILTER (WHERE status = 'dead')::bigint,
			COUNT(*) FILTER (WHERE status = 'completed')::bigint
		FROM outbox_jobs
	`).Scan(&s.Pending, &s.Processing, &s.Failed, &s.Dead, &s.Completed)
	return s, err
}

func truncateErr(s string) string {
	const max = 2000
	if len(s) <= max {
		return s
	}
	return s[:max]
}

var _ appoutbox.Repository = (*OutboxRepository)(nil)
