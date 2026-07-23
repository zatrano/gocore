package notification

import (
	"context"

	"github.com/zatrano/gocore/pkg/pagination"
)

// Repository, uygulama içi bildirimler için persistence portudur (Dependency
// Inversion). PostgreSQL implementasyonu adapters katmanındadır.
type Repository interface {
	// Save, yeni bir bildirimi kalıcılaştırır.
	Save(ctx context.Context, n *Notification) error

	// ListByRecipient, bir alıcının bildirimlerini sayfa tabanlı sayfalama ile
	// en yeni önce döner. unreadOnly true ise yalnızca okunmamışlar listelenir.
	ListByRecipient(ctx context.Context, recipientID string, page pagination.Request, unreadOnly bool) (pagination.Page[*Notification], error)

	// MarkRead, alıcıya ait bir bildirimi okundu işaretler. Kayıt yoksa veya
	// başka kullanıcıya aitse ErrNotFound döner (sahiplik kontrolü).
	MarkRead(ctx context.Context, id ID, recipientID string) error

	// MarkAllRead, alıcının tüm okunmamış bildirimlerini okundu işaretler.
	// Etkilenen satır sayısını döner (0 geçerli — zaten hepsi okunmuş olabilir).
	MarkAllRead(ctx context.Context, recipientID string) (int64, error)

	// Delete, alıcıya ait bir bildirimi siler. Kayıt yoksa veya başka kullanıcıya
	// aitse ErrNotFound döner.
	Delete(ctx context.Context, id ID, recipientID string) error

	// DeleteAll, alıcının tüm bildirimlerini siler; silinen satır sayısını döner.
	DeleteAll(ctx context.Context, recipientID string) (int64, error)

	// CountUnread, alıcının okunmamış bildirim sayısını döner.
	CountUnread(ctx context.Context, recipientID string) (int64, error)
}
