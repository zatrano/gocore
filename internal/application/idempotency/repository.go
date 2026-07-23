package idempotency

import (
	"context"
	"time"
)

// Status, idempotency kaydı durumudur.
type Status string

const (
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

// Record, kalıcı idempotency kaydıdır.
type Record struct {
	ID          string
	Scope       string
	Key         string
	ActorID     string
	RequestHash string
	Status      Status
	Response    []byte
	ExpiresAt   time.Time
}

// Repository, idempotency kayıtlarını saklar.
type Repository interface {
	Insert(ctx context.Context, rec Record) error
	Find(ctx context.Context, scope, key, actorID string) (Record, error)
	Complete(ctx context.Context, id string, response []byte) error
	Fail(ctx context.Context, id string) error
}
