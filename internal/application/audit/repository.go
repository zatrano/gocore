package audit

import (
	"context"
	"time"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Log, denetim kaydı okuma modelidir.
type Log struct {
	ID            string
	ActorID       string
	ActorEmail    string
	ActorType     string
	Action        string
	Resource      string
	ResourceID    string
	IP            string
	UserAgent     string
	Source        string
	CorrelationID string
	Metadata      map[string]any
	OccurredAt    time.Time
	CreatedAt     time.Time
}

// ListFilter, denetim kaydı listeleme filtreleridir.
type ListFilter struct {
	Action   string
	Resource string
	Actor    string // actor_id veya e-posta parçası
}

// Repository, denetim kayıtlarını okuma portudur.
type Repository interface {
	List(ctx context.Context, filter ListFilter, page pagination.Request) (pagination.Page[Log], error)
	FindByID(ctx context.Context, id string) (Log, error)
}
