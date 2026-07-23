package contact

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Service, iletişim formu use-case'lerinin yüzeyi (facade).
type Service struct {
	submit   *SubmitHandler
	list     *ListHandler
	get      *GetHandler
	markRead *MarkReadHandler
}

// ServiceDeps, Service bağımlılıklarını gruplar.
type ServiceDeps struct {
	Submit   *SubmitHandler
	List     *ListHandler
	Get      *GetHandler
	MarkRead *MarkReadHandler
}

// NewService, iletişim facade'ini kurar.
func NewService(d ServiceDeps) *Service {
	return &Service{
		submit: d.Submit, list: d.List, get: d.Get, markRead: d.MarkRead,
	}
}

func (s *Service) Submit(ctx context.Context, cmd SubmitCommand) (View, error) {
	return s.submit.Handle(ctx, cmd)
}

func (s *Service) List(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	return s.list.Handle(ctx, q)
}

func (s *Service) Get(ctx context.Context, id string) (View, error) {
	return s.get.Handle(ctx, id)
}

func (s *Service) MarkRead(ctx context.Context, cmd MarkReadCommand) (View, error) {
	return s.markRead.Handle(ctx, cmd)
}
