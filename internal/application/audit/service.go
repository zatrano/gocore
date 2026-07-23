package audit

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Service, denetim kaydı use-case'lerinin yüzeyi (facade).
type Service struct {
	list *ListHandler
	get  *GetHandler
}

// ServiceDeps, Service bağımlılıklarını gruplar.
type ServiceDeps struct {
	List *ListHandler
	Get  *GetHandler
}

// NewService, denetim facade'ini kurar.
func NewService(d ServiceDeps) *Service {
	return &Service{list: d.List, get: d.Get}
}

func (s *Service) List(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	return s.list.Handle(ctx, q)
}

func (s *Service) Get(ctx context.Context, id string) (View, error) {
	return s.get.Handle(ctx, id)
}
