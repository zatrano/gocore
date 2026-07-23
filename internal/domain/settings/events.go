package settings

import "github.com/zatrano/gocore/internal/domain/shared"

const (
	EventSMSProviderChanged     = "settings.sms_provider_changed"
	EventPaymentProviderChanged = "settings.payment_provider_changed"
)

// SMSProviderChangedEvent, aktif SMS sağlayıcısı değiştiğinde üretilir.
type SMSProviderChangedEvent struct {
	shared.BaseEvent
	OldProvider string
	NewProvider string
}

// PaymentProviderChangedEvent, aktif ödeme sağlayıcısı değiştiğinde üretilir.
type PaymentProviderChangedEvent struct {
	shared.BaseEvent
	OldProvider string
	NewProvider string
}
