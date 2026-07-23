package audit

import (
	"context"

	"github.com/google/uuid"
)

// GetHandler, tek denetim kaydı sorgusunu işler.
type GetHandler struct {
	repo Repository
}

// NewGetHandler, GetHandler'ı kurar.
func NewGetHandler(repo Repository) *GetHandler {
	return &GetHandler{repo: repo}
}

// Handle, kimliğe göre denetim kaydını döner.
func (h *GetHandler) Handle(ctx context.Context, id string) (View, error) {
	if _, err := uuid.Parse(id); err != nil {
		return View{}, ErrNotFound
	}
	log, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return View{}, err
	}
	return toView(log), nil
}
