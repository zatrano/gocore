package contact

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Repository, iletişim mesajı kalıcılık portudur.
type Repository interface {
	Save(ctx context.Context, m *Message) error
	FindByID(ctx context.Context, id ID) (*Message, error)
	List(ctx context.Context, page pagination.Request, unreadOnly bool) (pagination.Page[*Message], error)
	MarkRead(ctx context.Context, id ID) error
}
