package payment

import (
	"context"
	"time"

	"github.com/zatrano/gocore/pkg/pagination"
)

// ListFilter, ödeme listeleme filtreleridir.
type ListFilter struct {
	Status   string
	Provider string
}

// Repository, ödeme kayıtları için persistence portudur.
type Repository interface {
	Save(ctx context.Context, p *Payment) error
	FindByReference(ctx context.Context, reference string) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
	List(ctx context.Context, filter ListFilter, page pagination.Request) (pagination.Page[*Payment], error)
	// ListReconcileCandidates, uzun süredir bekleyen ve sağlayıcıyla hizalanması gereken kayıtları döner.
	ListReconcileCandidates(ctx context.Context, minAge time.Duration, limit int) ([]*Payment, error)
}
