package upload

import "github.com/zatrano/gocore/internal/domain/shared"

const EventBatchCompleted = "upload.batch_completed"

// MIMEStat, batch içindeki MIME türü özetidir.
type MIMEStat struct {
	MIME  string `json:"mime"`
	Count int    `json:"count"`
	Bytes int64  `json:"bytes"`
}

// BatchCompletedEvent, çoklu yükleme tamamlandığında yayınlanır.
type BatchCompletedEvent struct {
	shared.BaseEvent
	Total    int        `json:"total"`
	Accepted int        `json:"accepted"`
	Rejected int        `json:"rejected"`
	MIMES    []MIMEStat `json:"mime_summary,omitempty"`
}

func NewBatchCompletedEvent(batchID string, total, accepted, rejected int, mimeSummary []MIMEStat) BatchCompletedEvent {
	return BatchCompletedEvent{
		BaseEvent: shared.NewBaseEvent(EventBatchCompleted, batchID),
		Total:     total,
		Accepted:  accepted,
		Rejected:  rejected,
		MIMES:     mimeSummary,
	}
}

func (e BatchCompletedEvent) EventName() string { return e.BaseEvent.EventName() }
