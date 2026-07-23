package payment

import "github.com/zatrano/gocore/internal/domain/shared"

// ProviderName, desteklenen ödeme sağlayıcı kodlarıdır.
type ProviderName string

const (
	ProviderIyzico ProviderName = "iyzico"
	ProviderMoka   ProviderName = "moka"
)

func (p ProviderName) String() string { return string(p) }

const (
	EventThreeDSInitialized    = "payment.threeds.initialized"
	EventThreeDSCompleted      = "payment.threeds.completed"
	EventThreeDSFailed         = "payment.threeds.failed"
	EventThreeDSWebhookApplied = "payment.threeds.webhook_applied"
	EventThreeDSReconciled     = "payment.threeds.reconciled"
)

// ThreeDSEventMeta, ödeme domain event'lerinde paylaşılan güvenli metadata alanlarıdır.
type ThreeDSEventMeta struct {
	Reference string
	Provider  string
	Status    string
	Stage     string
	Amount    string
	Currency  string
}

func threeDSEventMeta(p *Payment) ThreeDSEventMeta {
	return ThreeDSEventMeta{
		Reference: p.Reference(),
		Provider:  p.Provider(),
		Status:    string(p.Status()),
		Stage:     string(p.Stage()),
		Amount:    p.Amount(),
		Currency:  p.Currency(),
	}
}

// ThreeDSInitialized, 3DS ödemesi başlatıldığında yayınlanır.
type ThreeDSInitialized struct {
	shared.BaseEvent
	ThreeDSEventMeta
}

func NewThreeDSInitialized(p *Payment) ThreeDSInitialized {
	return ThreeDSInitialized{
		BaseEvent:        shared.NewBaseEvent(EventThreeDSInitialized, p.ID()),
		ThreeDSEventMeta: threeDSEventMeta(p),
	}
}

func (e ThreeDSInitialized) EventName() string { return e.BaseEvent.EventName() }

// ThreeDSCompleted, 3DS ödeme başarıyla tamamlandığında yayınlanır.
type ThreeDSCompleted struct {
	shared.BaseEvent
	ThreeDSEventMeta
}

func NewThreeDSCompleted(p *Payment) ThreeDSCompleted {
	return ThreeDSCompleted{
		BaseEvent:        shared.NewBaseEvent(EventThreeDSCompleted, p.ID()),
		ThreeDSEventMeta: threeDSEventMeta(p),
	}
}

func (e ThreeDSCompleted) EventName() string { return e.BaseEvent.EventName() }

// ThreeDSFailed, 3DS ödeme başarısız olduğunda yayınlanır.
type ThreeDSFailed struct {
	shared.BaseEvent
	ThreeDSEventMeta
}

func NewThreeDSFailed(p *Payment) ThreeDSFailed {
	return ThreeDSFailed{
		BaseEvent:        shared.NewBaseEvent(EventThreeDSFailed, p.ID()),
		ThreeDSEventMeta: threeDSEventMeta(p),
	}
}

func (e ThreeDSFailed) EventName() string { return e.BaseEvent.EventName() }

// ThreeDSWebhookApplied, iyzico webhook ile oturum güncellendiğinde yayınlanır.
type ThreeDSWebhookApplied struct {
	shared.BaseEvent
	ThreeDSEventMeta
}

func NewThreeDSWebhookApplied(p *Payment) ThreeDSWebhookApplied {
	return ThreeDSWebhookApplied{
		BaseEvent:        shared.NewBaseEvent(EventThreeDSWebhookApplied, p.ID()),
		ThreeDSEventMeta: threeDSEventMeta(p),
	}
}

func (e ThreeDSWebhookApplied) EventName() string { return e.BaseEvent.EventName() }

// ThreeDSReconciled, bekleyen ödeme sağlayıcıyla hizalandığında yayınlanır.
type ThreeDSReconciled struct {
	shared.BaseEvent
	ThreeDSEventMeta
}

func NewThreeDSReconciled(p *Payment) ThreeDSReconciled {
	return ThreeDSReconciled{
		BaseEvent:        shared.NewBaseEvent(EventThreeDSReconciled, p.ID()),
		ThreeDSEventMeta: threeDSEventMeta(p),
	}
}

func (e ThreeDSReconciled) EventName() string { return e.BaseEvent.EventName() }
