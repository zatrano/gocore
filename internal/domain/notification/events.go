package notification

import "github.com/zatrano/gocore/internal/domain/shared"

// Event adları.
const (
	EventCreated            = "notification.created"
	EventRead               = "notification.read"
	EventManualSendAccepted = "notification.manual_send_accepted"
	EventBulkSendAccepted   = "notification.bulk_send_accepted"
)

// CreatedEvent, yeni bir uygulama içi bildirim oluşturulduğunda üretilir.
type CreatedEvent struct {
	shared.BaseEvent
	RecipientID string
}

// ReadEvent, bir bildirim okundu olarak işaretlendiğinde üretilir.
type ReadEvent struct {
	shared.BaseEvent
}

// ManualSendAcceptedEvent, tekil elle gönderim kabul edildiğinde üretilir.
type ManualSendAcceptedEvent struct {
	shared.BaseEvent
	Channel  string
	Accepted int
}

func NewManualSendAcceptedEvent(aggregateID, channel string) ManualSendAcceptedEvent {
	return ManualSendAcceptedEvent{
		BaseEvent: shared.NewBaseEvent(EventManualSendAccepted, aggregateID),
		Channel:   channel,
		Accepted:  1,
	}
}

func (e ManualSendAcceptedEvent) EventName() string { return e.BaseEvent.EventName() }

// BulkSendAcceptedEvent, toplu elle gönderim kabul edildiğinde üretilir.
type BulkSendAcceptedEvent struct {
	shared.BaseEvent
	Channel      string
	Total        int
	Accepted     int
	InvalidCount int
}

func NewBulkSendAcceptedEvent(aggregateID, channel string, total, accepted, invalidCount int) BulkSendAcceptedEvent {
	return BulkSendAcceptedEvent{
		BaseEvent:    shared.NewBaseEvent(EventBulkSendAccepted, aggregateID),
		Channel:      channel,
		Total:        total,
		Accepted:     accepted,
		InvalidCount: invalidCount,
	}
}

func (e BulkSendAcceptedEvent) EventName() string { return e.BaseEvent.EventName() }
