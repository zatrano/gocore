package user

import (
	"context"

	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/pagination"
)

// --- Query girdileri (immutable DTO) ---

// GetQuery, tek kullanıcı getirme girdisidir.
type GetQuery struct {
	UserID    string
	ActorID   string
	ActorRole string
}

// ListQuery, kullanıcı listeleme girdisidir (filtreleme + sayfalama).
type ListQuery struct {
	ActorRole string
	Role      string // "" = filtre yok
	Active    *bool  // nil = filtre yok
	Search    string
	Deleted   string // "" = canlı, "only" = silinenler, "all" = hepsi
	Page      int
	Limit     int
	Ascending bool
	Cursor    string // opak keyset imleci; doluysa Page yok sayılır
}

// GetHandler, tek kullanıcı sorgusunu işler (read-side).
type GetHandler struct {
	repo   user.Repository
	access Access
}

// NewGetHandler, GetHandler'ı kurar.
func NewGetHandler(repo user.Repository, access Access) *GetHandler {
	return &GetHandler{repo: repo, access: access}
}

// Handle, kullanıcıyı kimliğine göre getirir. Kendi profili veya users:read
// izni gerekir.
func (h *GetHandler) Handle(ctx context.Context, q GetQuery) (View, error) {
	if err := h.access.CanReadUser(ctx, q.ActorRole, q.ActorID, q.UserID); err != nil {
		return View{}, err
	}
	id, err := user.ParseID(q.UserID)
	if err != nil {
		return View{}, err
	}
	u, err := h.repo.FindByIDIncludeDeleted(ctx, id)
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// ListHandler, kullanıcı listeleme sorgusunu işler.
type ListHandler struct {
	repo   user.Repository
	access Access
}

// NewListHandler, ListHandler'ı kurar.
func NewListHandler(repo user.Repository, access Access) *ListHandler {
	return &ListHandler{repo: repo, access: access}
}

// Handle, filtrelenmiş ve sayfalanmış kullanıcı listesini döner.
// users:list izni gerekir.
func (h *ListHandler) Handle(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	if err := h.access.CanListUsers(ctx, q.ActorRole); err != nil {
		return pagination.Page[View]{}, err
	}

	filter := user.ListFilter{Active: q.Active, Search: q.Search, Deleted: q.Deleted}
	if q.Role != "" {
		role, err := user.ParseRole(q.Role)
		if err != nil {
			return pagination.Page[View]{}, err
		}
		filter.Role = &role
	}

	pageReq := pagination.Request{
		Page:      q.Page,
		Limit:     q.Limit,
		Ascending: q.Ascending,
		Cursor:    q.Cursor,
	}
	if q.Cursor != "" {
		pageReq.Page = 1
	}

	page, err := h.repo.List(ctx, filter, pageReq)
	if err != nil {
		return pagination.Page[View]{}, err
	}

	views := newViews(page.Items)
	out := pagination.NewPage(views, page.Page, page.Limit, page.Total)
	out.NextCursor = page.NextCursor
	return out, nil
}
