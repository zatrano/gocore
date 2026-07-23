package notification

import (
	"context"

	dnotif "github.com/zatrano/gocore/internal/domain/notification"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
)

// View, bir uygulama içi bildirimin okuma-tarafı (read model) temsilidir.
type View struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Content   string             `json:"content"`
	Read      bool               `json:"read"`
	CreatedAt datetime.JSONTime  `json:"created_at"`
	ReadAt    *datetime.JSONTime `json:"read_at,omitempty"`
}

func newView(n *dnotif.Notification) View {
	return View{
		ID:        n.ID().String(),
		Title:     n.Title(),
		Content:   n.Content(),
		Read:      n.IsRead(),
		CreatedAt: datetime.FromTime(n.CreatedAt()),
		ReadAt:    datetime.PtrFromTime(n.ReadAt()),
	}
}

// ListMyQuery, kimliği doğrulanmış kullanıcının bildirimlerini listeler.
type ListMyQuery struct {
	UserID     string
	Page       int
	Limit      int
	UnreadOnly bool
}

// ListHandler, kullanıcının bildirimlerini keyset sayfalama ile döner.
type ListHandler struct {
	repo dnotif.Repository
}

// NewListHandler, ListHandler'ı kurar.
func NewListHandler(repo dnotif.Repository) *ListHandler { return &ListHandler{repo: repo} }

// Handle, bildirim listesini View DTO'suna projekte ederek döner.
func (h *ListHandler) Handle(ctx context.Context, q ListMyQuery) (pagination.Page[View], error) {
	page, err := h.repo.ListByRecipient(ctx, q.UserID, pagination.Request{
		Page: q.Page, Limit: q.Limit,
	}, q.UnreadOnly)
	if err != nil {
		return pagination.Page[View]{}, err
	}

	views := make([]View, 0, len(page.Items))
	for _, n := range page.Items {
		views = append(views, newView(n))
	}
	return pagination.NewPage(views, page.Page, page.Limit, page.Total), nil
}

// UnreadCountHandler, okunmamış bildirim sayısını döner.
type UnreadCountHandler struct {
	repo dnotif.Repository
}

// NewUnreadCountHandler, UnreadCountHandler'ı kurar.
func NewUnreadCountHandler(repo dnotif.Repository) *UnreadCountHandler {
	return &UnreadCountHandler{repo: repo}
}

// Handle, kullanıcının okunmamış bildirim sayısını döner.
func (h *UnreadCountHandler) Handle(ctx context.Context, userID string) (int64, error) {
	return h.repo.CountUnread(ctx, userID)
}
