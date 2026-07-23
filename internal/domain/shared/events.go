package shared

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent, bir aggregate içinde gerçekleşen ve dış dünyaya (event-driven
// tasarım) yayınlanabilecek önemli bir olguyu temsil eder.
type DomainEvent interface {
	// EventID, olayın benzersiz kimliği (idempotent audit/outbox için).
	EventID() string
	// EventName, olayın benzersiz adı (ör. "user.registered").
	EventName() string
	// OccurredAt, olayın gerçekleşme zamanı.
	OccurredAt() time.Time
	// AggregateID, olayın ait olduğu aggregate'in kimliği.
	AggregateID() string
}

// BaseEvent, DomainEvent implementasyonları için ortak alanları sağlar.
type BaseEvent struct {
	ID           string
	Name         string
	At           time.Time
	AggregateID_ string
}

func (e BaseEvent) EventID() string       { return e.ID }
func (e BaseEvent) EventName() string     { return e.Name }
func (e BaseEvent) OccurredAt() time.Time { return e.At }
func (e BaseEvent) AggregateID() string   { return e.AggregateID_ }

// NewBaseEvent, verilen ad ve aggregate id ile temel event üretir.
func NewBaseEvent(name, aggregateID string) BaseEvent {
	return BaseEvent{
		ID:           uuid.New().String(),
		Name:         name,
		At:           time.Now().UTC(),
		AggregateID_: aggregateID,
	}
}

// EventRecorder, aggregate root'ların ürettiği olayları biriktirmesini sağlar.
// Uygulama katmanı, işlem tamamlandığında bu olayları toplar ve yayınlar.
type EventRecorder struct {
	events []DomainEvent
}

// Record, yeni bir domain event kaydeder.
func (r *EventRecorder) Record(e DomainEvent) {
	r.events = append(r.events, e)
}

// PullEvents, biriken olayları döner ve tamponu temizler.
func (r *EventRecorder) PullEvents() []DomainEvent {
	events := r.events
	r.events = nil
	return events
}
