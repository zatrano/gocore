package notification

import (
	"context"

	dnotif "github.com/zatrano/gocore/internal/domain/notification"
)

// MarkReadCommand, bir bildirimi okundu işaretleme girdisidir. UserID sahiplik
// kontrolü için zorunludur (kullanıcı yalnızca kendi bildirimini okuyabilir).
type MarkReadCommand struct {
	UserID         string
	NotificationID string
}

// MarkReadHandler, bildirimi okundu işaretleme command'ini işler.
type MarkReadHandler struct {
	repo dnotif.Repository
}

// NewMarkReadHandler, MarkReadHandler'ı kurar.
func NewMarkReadHandler(repo dnotif.Repository) *MarkReadHandler {
	return &MarkReadHandler{repo: repo}
}

// Handle, bildirimi okundu işaretler.
func (h *MarkReadHandler) Handle(ctx context.Context, cmd MarkReadCommand) error {
	id, err := dnotif.ParseID(cmd.NotificationID)
	if err != nil {
		return err
	}
	return h.repo.MarkRead(ctx, id, cmd.UserID)
}

// MarkAllReadCommand, kullanıcının tüm bildirimlerini okundu işaretleme girdisidir.
type MarkAllReadCommand struct {
	UserID string
}

// MarkAllReadHandler, tüm okunmamış bildirimleri okundu işaretler.
type MarkAllReadHandler struct {
	repo dnotif.Repository
}

// NewMarkAllReadHandler, MarkAllReadHandler'ı kurar.
func NewMarkAllReadHandler(repo dnotif.Repository) *MarkAllReadHandler {
	return &MarkAllReadHandler{repo: repo}
}

// Handle, alıcının tüm okunmamış bildirimlerini okundu işaretler; etkilenen sayıyı döner.
func (h *MarkAllReadHandler) Handle(ctx context.Context, cmd MarkAllReadCommand) (int64, error) {
	return h.repo.MarkAllRead(ctx, cmd.UserID)
}

// DeleteCommand, bir bildirimi silme girdisidir.
type DeleteCommand struct {
	UserID         string
	NotificationID string
}

// DeleteHandler, bildirimi siler.
type DeleteHandler struct {
	repo dnotif.Repository
}

// NewDeleteHandler, DeleteHandler'ı kurar.
func NewDeleteHandler(repo dnotif.Repository) *DeleteHandler {
	return &DeleteHandler{repo: repo}
}

// Handle, bildirimi siler.
func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	id, err := dnotif.ParseID(cmd.NotificationID)
	if err != nil {
		return err
	}
	return h.repo.Delete(ctx, id, cmd.UserID)
}

// DeleteAllCommand, kullanıcının tüm bildirimlerini silme girdisidir.
type DeleteAllCommand struct {
	UserID string
}

// DeleteAllHandler, tüm bildirimleri siler.
type DeleteAllHandler struct {
	repo dnotif.Repository
}

// NewDeleteAllHandler, DeleteAllHandler'ı kurar.
func NewDeleteAllHandler(repo dnotif.Repository) *DeleteAllHandler {
	return &DeleteAllHandler{repo: repo}
}

// Handle, alıcının tüm bildirimlerini siler; silinen sayıyı döner.
func (h *DeleteAllHandler) Handle(ctx context.Context, cmd DeleteAllCommand) (int64, error) {
	return h.repo.DeleteAll(ctx, cmd.UserID)
}
