// Package notification, uygulama içi (in-app) bildirim bounded context'idir.
// SMS ve e-posta gibi geçici kanalların aksine, uygulama içi bildirimler kalıcı
// olarak saklanır ve kullanıcı tarafından listelenip okundu işaretlenebilir.
package notification

import (
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// Notification, uygulama içi bildirim aggregate root'udur. Belirli bir alıcıya
// (kullanıcıya) ait başlık + içerik taşır ve okundu durumunu yönetir.
type Notification struct {
	shared.EventRecorder

	id          ID
	recipientID string // hedef kullanıcı kimliği (user.ID string temsili)
	title       string
	content     string
	read        bool
	createdAt   time.Time
	readAt      *time.Time
}

// New, yeni bir uygulama içi bildirim oluşturur ve CreatedEvent üretir.
// title/content, çağıran tarafından (i18n render sonrası) hazır metin olarak verilir.
func New(recipientID, title, content string) (*Notification, error) {
	recipientID = strings.TrimSpace(recipientID)
	if recipientID == "" {
		return nil, ErrRecipientRequired
	}
	if strings.TrimSpace(title) == "" {
		return nil, ErrTitleRequired
	}
	if strings.TrimSpace(content) == "" {
		return nil, ErrContentRequired
	}

	n := &Notification{
		id:          NewID(),
		recipientID: recipientID,
		title:       title,
		content:     content,
		read:        false,
		createdAt:   time.Now().UTC(),
	}
	n.Record(CreatedEvent{
		BaseEvent:   shared.NewBaseEvent(EventCreated, n.id.String()),
		RecipientID: recipientID,
	})
	return n, nil
}

// Hydrate, persistence katmanından okunan verilerle bir Notification'ı yeniden
// oluşturur. Yalnızca repository tarafından kullanılmalıdır (event üretmez).
func Hydrate(id ID, recipientID, title, content string, read bool, createdAt time.Time, readAt *time.Time) *Notification {
	return &Notification{
		id:          id,
		recipientID: recipientID,
		title:       title,
		content:     content,
		read:        read,
		createdAt:   createdAt,
		readAt:      readAt,
	}
}

// MarkRead, bildirimi okundu olarak işaretler (idempotent). Zaten okunmuşsa
// event üretmez.
func (n *Notification) MarkRead() {
	if n.read {
		return
	}
	now := time.Now().UTC()
	n.read = true
	n.readAt = &now
	n.Record(ReadEvent{BaseEvent: shared.NewBaseEvent(EventRead, n.id.String())})
}

// --- Getter'lar ---

func (n *Notification) ID() ID               { return n.id }
func (n *Notification) RecipientID() string  { return n.recipientID }
func (n *Notification) Title() string        { return n.title }
func (n *Notification) Content() string      { return n.content }
func (n *Notification) IsRead() bool         { return n.read }
func (n *Notification) CreatedAt() time.Time { return n.createdAt }
func (n *Notification) ReadAt() *time.Time   { return n.readAt }
