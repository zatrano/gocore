package contact

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// Message, iletişim formu aggregate'idir.
type Message struct {
	id        ID
	name      string
	email     Email
	body      string
	locale    string
	ip        string
	userAgent string
	status    Status
	createdAt time.Time
	readAt    *time.Time
	events    shared.EventRecorder
}

// ID, iletişim mesajı kimliği.
type ID struct{ value uuid.UUID }

func (id ID) String() string { return id.value.String() }

// ParseID, string kimliği parse eder.
func ParseID(s string) (ID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return ID{}, ErrInvalidID
	}
	return ID{value: u}, nil
}

// Email, iletişim formu e-postası.
type Email struct{ value string }

func (e Email) String() string { return e.value }

// NewEmail, e-posta doğrular.
func NewEmail(s string) (Email, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || !strings.Contains(s, "@") || len(s) > 254 {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: s}, nil
}

// Status, mesaj işleme durumu.
type Status string

const (
	StatusReceived Status = "received"
	StatusQueued   Status = "queued"
	StatusNotified Status = "notified"
	StatusFailed   Status = "failed"
)

// Submit, yeni iletişim mesajı oluşturur.
func Submit(name, email, body, locale, ip, userAgent string) (*Message, error) {
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	locale = strings.TrimSpace(locale)
	if locale == "" {
		locale = "tr"
	}
	if len(name) < 2 || len(name) > 100 {
		return nil, ErrInvalidName
	}
	if len(body) < 5 || len(body) > 2000 {
		return nil, ErrInvalidMessage
	}
	em, err := NewEmail(email)
	if err != nil {
		return nil, err
	}
	m := &Message{
		id:        ID{value: uuid.New()},
		name:      name,
		email:     em,
		body:      body,
		locale:    locale,
		ip:        strings.TrimSpace(ip),
		userAgent: strings.TrimSpace(userAgent),
		status:    StatusReceived,
		createdAt: time.Now().UTC(),
	}
	m.events.Record(SubmittedEvent{
		BaseEvent: shared.NewBaseEvent(EventSubmitted, m.id.String()),
		Name:      m.name,
		Email:     m.email.String(),
	})
	return m, nil
}

func (m *Message) ID() ID               { return m.id }
func (m *Message) Name() string         { return m.name }
func (m *Message) Email() Email         { return m.email }
func (m *Message) Body() string         { return m.body }
func (m *Message) Locale() string       { return m.locale }
func (m *Message) IP() string           { return m.ip }
func (m *Message) UserAgent() string    { return m.userAgent }
func (m *Message) Status() Status       { return m.status }
func (m *Message) CreatedAt() time.Time { return m.createdAt }
func (m *Message) ReadAt() *time.Time   { return m.readAt }
func (m *Message) IsRead() bool         { return m.readAt != nil }

// MarkQueued, e-posta işi kuyruğa alındığında durumu günceller.
func (m *Message) MarkQueued() { m.status = StatusQueued }

// MarkRead, mesajı okundu olarak işaretler (idempotent).
func (m *Message) MarkRead() {
	if m.readAt != nil {
		return
	}
	now := time.Now().UTC()
	m.readAt = &now
}

// PullEvents, biriken olayları döner.
func (m *Message) PullEvents() []shared.DomainEvent { return m.events.PullEvents() }

// Reconstitute, kalıcılıktan aggregate yeniden oluşturur.
func Reconstitute(
	id ID, name string, email Email, body, locale, ip, userAgent string,
	status Status, createdAt time.Time, readAt *time.Time,
) *Message {
	return &Message{
		id: id, name: name, email: email, body: body, locale: locale,
		ip: ip, userAgent: userAgent, status: status, createdAt: createdAt, readAt: readAt,
	}
}
