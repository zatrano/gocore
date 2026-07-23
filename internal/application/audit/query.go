package audit

import (
	"context"

	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
)

// View, denetim kaydı listeleme DTO'sudur.
type View struct {
	ID            string         `json:"id"`
	ActorID       string         `json:"actor_id,omitempty"`
	ActorEmail    string         `json:"actor_email,omitempty"`
	ActorType     string         `json:"actor_type,omitempty"`
	Action        string         `json:"action"`
	Resource      string         `json:"resource"`
	ResourceID    string         `json:"resource_id,omitempty"`
	IP            string         `json:"ip,omitempty"`
	UserAgent     string         `json:"user_agent,omitempty"`
	Source        string         `json:"source,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	ChangeSummary string         `json:"change_summary,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

// ListQuery, denetim kaydı listeleme girdisidir.
type ListQuery struct {
	Action    string
	Resource  string
	Actor     string
	Page      int
	Limit     int
	Ascending bool
}

// ListHandler, denetim kayıtlarını listeler.
type ListHandler struct {
	repo Repository
}

// NewListHandler, ListHandler'ı kurar.
func NewListHandler(repo Repository) *ListHandler {
	return &ListHandler{repo: repo}
}

// Handle, filtrelenmiş denetim kayıtlarını sayfalar.
func (h *ListHandler) Handle(ctx context.Context, q ListQuery) (pagination.Page[View], error) {
	page, err := h.repo.List(ctx, ListFilter{
		Action: q.Action, Resource: q.Resource, Actor: q.Actor,
	}, pagination.Request{Page: q.Page, Limit: q.Limit, Ascending: q.Ascending})
	if err != nil {
		return pagination.Page[View]{}, err
	}
	views := make([]View, 0, len(page.Items))
	for _, item := range page.Items {
		views = append(views, toView(item))
	}
	return pagination.NewPage(views, page.Page, page.Limit, page.Total), nil
}

func toView(l Log) View {
	return View{
		ID: l.ID, ActorID: l.ActorID, ActorEmail: l.ActorEmail, ActorType: l.ActorType,
		Action: l.Action, Resource: l.Resource, ResourceID: l.ResourceID,
		IP: l.IP, UserAgent: l.UserAgent, Source: l.Source, CorrelationID: l.CorrelationID,
		Metadata: l.Metadata, ChangeSummary: formatChangeSummary(l.Action, l.Metadata),
		CreatedAt: datetime.FormatDateTime(l.CreatedAt),
	}
}
