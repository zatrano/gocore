// Package outbox, PostgreSQL tabanlı kalıcı iş kuyruğu portlarını tanımlar.
package outbox

import (
	"context"
	"encoding/json"
	"time"
)

// Job türleri.
const (
	KindDomainEvent          = "domain.event"
	KindEmailSend            = "email.send"
	KindNotificationDispatch = "notification.dispatch"
)

// Status değerleri.
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusDead       = "dead"
)

// Job, kuyruğa yazılacak veya claim edilen iş birimidir.
type Job struct {
	ID             string
	Kind           string
	AggregateType  string
	AggregateID    string
	IdempotencyKey string
	Payload        json.RawMessage
	Status         string
	Attempts       int
	MaxAttempts    int
	AvailableAt    time.Time
	LeaseUntil     *time.Time
	LastError      string
	CreatedAt      time.Time
	CompletedAt    *time.Time
}

// NewJob, yeni bir bekleyen iş oluşturur.
func NewJob(kind, aggregateType, aggregateID, idempotencyKey string, payload any) (Job, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return Job{}, err
	}
	return Job{
		Kind:           kind,
		AggregateType:  aggregateType,
		AggregateID:    aggregateID,
		IdempotencyKey: idempotencyKey,
		Payload:        b,
		Status:         StatusPending,
		MaxAttempts:    8,
		AvailableAt:    time.Now().UTC(),
	}, nil
}

// Enqueuer, aynı transaction içinde (veya dışında) iş kuyruğuna yazar.
type Enqueuer interface {
	Enqueue(ctx context.Context, job Job) error
}

// Repository, claim / tamamlama / yeniden deneme işlemlerini sağlar.
type Repository interface {
	Enqueuer
	Claim(ctx context.Context, limit int, lease time.Duration) ([]Job, error)
	MarkCompleted(ctx context.Context, id string) error
	MarkRetryable(ctx context.Context, id string, attempts int, nextAttempt time.Time, lastErr string) error
	MarkDead(ctx context.Context, id string, attempts int, lastErr string) error
	Stats(ctx context.Context) (Stats, error)
}

// Stats, kuyruk gözlemlenebilirlik özetidir.
type Stats struct {
	Pending    int64
	Processing int64
	Failed     int64
	Dead       int64
	Completed  int64
}

// EmailPayload, kind=email.send için gövde.
type EmailPayload struct {
	To       []string `json:"to"`
	Subject  string   `json:"subject"`
	HTMLBody string   `json:"html_body,omitempty"`
	TextBody string   `json:"text_body,omitempty"`
	From     string   `json:"from,omitempty"`
	ReplyTo  string   `json:"reply_to,omitempty"`
}

// DispatchPayload, kind=notification.dispatch için gövde.
type DispatchPayload struct {
	Channel          string   `json:"channel"`
	UserID           string   `json:"user_id,omitempty"`
	Email            string   `json:"email,omitempty"`
	Phone            string   `json:"phone,omitempty"`
	Locale           string   `json:"locale,omitempty"`
	TitleKey         string   `json:"title_key,omitempty"`
	ContentKey       string   `json:"content_key,omitempty"`
	HTMLContentKey   string   `json:"html_content_key,omitempty"`
	TitleFallback    string   `json:"title_fallback,omitempty"`
	BodyFallback     string   `json:"body_fallback,omitempty"`
	HTMLBodyFallback string   `json:"html_body_fallback,omitempty"`
	Args             []string `json:"args,omitempty"`
	LiteralTitle     string   `json:"literal_title,omitempty"`
	LiteralBody      string   `json:"literal_body,omitempty"`
	LiteralHTML      string   `json:"literal_html,omitempty"`
}

// AggregateTypeFromEvent, event adından aggregate türünü çıkarır (user.registered → user).
func AggregateTypeFromEvent(eventName string) string {
	for i := 0; i < len(eventName); i++ {
		if eventName[i] == '.' {
			return eventName[:i]
		}
	}
	return eventName
}

// ResourceFromEvent, audit resource adını event adından üretir.
func ResourceFromEvent(eventName string) string {
	return AggregateTypeFromEvent(eventName)
}

// DomainEventPayload, kind=domain.event için gövde.
type DomainEventPayload struct {
	EventID       string          `json:"event_id"`
	EventName     string          `json:"event_name"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	ActorID       string          `json:"actor_id,omitempty"`
	ActorType     string          `json:"actor_type,omitempty"`
	ActorEmail    string          `json:"actor_email,omitempty"`
	Source        string          `json:"source,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	IP            string          `json:"ip,omitempty"`
	UserAgent     string          `json:"user_agent,omitempty"`
	Data          json.RawMessage `json:"data"`
}
