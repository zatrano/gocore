package contact

import (
	"context"

	domaincontact "github.com/zatrano/gocore/internal/domain/contact"
	"github.com/zatrano/gocore/pkg/pagination"
)

// ListQuery, iletişim mesajı listeleme girdisidir.
type ListQuery struct {
	UnreadOnly bool
	Page       int
	Limit      int
	Ascending  bool
}

// ListHandler, iletişim mesajlarını listeler.
type ListHandler struct {
	repo domaincontact.Repository
}

// NewListHandler, ListHandler'ı kurar.
func NewListHandler(repo domaincontact.Repository) *ListHandler {
	return &ListHandler{repo: repo}
}

// Handle, filtrelenmiş ve sayfalanmış mesaj listesini döner.
func (h *ListHandler) Handle(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	page, err := h.repo.List(ctx, pagination.Request{
		Page: q.Page, Limit: q.Limit, Ascending: q.Ascending,
	}, q.UnreadOnly)
	if err != nil {
		return pagination.Page[View]{}, err
	}
	return pagination.NewPage(toViews(page.Items), page.Page, page.Limit, page.Total), nil
}

// GetHandler, tek iletişim mesajı sorgusunu işler.
type GetHandler struct {
	repo domaincontact.Repository
}

// NewGetHandler, GetHandler'ı kurar.
func NewGetHandler(repo domaincontact.Repository) *GetHandler {
	return &GetHandler{repo: repo}
}

// Handle, kimliğe göre iletişim mesajını döner.
func (h *GetHandler) Handle(ctx context.Context, id string) (View, error) {
	parsed, err := domaincontact.ParseID(id)
	if err != nil {
		return View{}, err
	}
	msg, err := h.repo.FindByID(ctx, parsed)
	if err != nil {
		return View{}, err
	}
	return toView(msg), nil
}

// MarkReadCommand, mesajı okundu işaretleme girdisidir.
type MarkReadCommand struct {
	ID string
}

// MarkReadHandler, mesajı okundu işaretler.
type MarkReadHandler struct {
	repo domaincontact.Repository
}

// NewMarkReadHandler, MarkReadHandler'ı kurar.
func NewMarkReadHandler(repo domaincontact.Repository) *MarkReadHandler {
	return &MarkReadHandler{repo: repo}
}

// Handle, mesajı okundu olarak işaretler ve güncel görünümü döner.
func (h *MarkReadHandler) Handle(ctx context.Context, cmd MarkReadCommand) (View, error) {
	parsed, err := domaincontact.ParseID(cmd.ID)
	if err != nil {
		return View{}, err
	}
	if err := h.repo.MarkRead(ctx, parsed); err != nil {
		return View{}, err
	}
	msg, err := h.repo.FindByID(ctx, parsed)
	if err != nil {
		return View{}, err
	}
	return toView(msg), nil
}
