package settings

import "github.com/zatrano/gocore/internal/domain/shared"

// PlatformSettings, platform genelindeki entegrasyon seçimlerinin aggregate root'udur.
// Aktif SMS/ödeme sağlayıcıları gibi alanlar bu tek aggregate altında tutulur;
// SMS ve ödeme kavramları value object (SMSProvider, PaymentProvider) düzeyinde kalır.
type PlatformSettings struct {
	shared.EventRecorder

	smsActiveProvider     SMSProvider
	paymentActiveProvider PaymentProvider
}

// DefaultPlatformSettings, varsayılan platform ayarlarını döner.
func DefaultPlatformSettings() *PlatformSettings {
	return &PlatformSettings{
		smsActiveProvider:     ProviderNetgsm,
		paymentActiveProvider: ProviderIyzico,
	}
}

// RehydratePlatformSettings, persistence'tan okunan ham değerlerle aggregate'i yeniden oluşturur.
func RehydratePlatformSettings(smsRaw, paymentRaw string) *PlatformSettings {
	s := DefaultPlatformSettings()
	if smsRaw != "" {
		if p, err := ParseSMSProvider(smsRaw); err == nil {
			s.smsActiveProvider = p
		}
	}
	if paymentRaw != "" {
		if p, err := ParsePaymentProvider(paymentRaw); err == nil {
			s.paymentActiveProvider = p
		}
	}
	return s
}

// SetSMSActiveProvider, aktif SMS sağlayıcısını değiştirir.
func (s *PlatformSettings) SetSMSActiveProvider(provider SMSProvider) error {
	if !IsValidSMSProvider(provider) {
		return ErrInvalidSMSProvider
	}
	if s.smsActiveProvider == provider {
		return nil
	}
	old := s.smsActiveProvider
	s.smsActiveProvider = provider
	s.Record(SMSProviderChangedEvent{
		BaseEvent:   shared.NewBaseEvent(EventSMSProviderChanged, KeySMSActiveProvider.String()),
		OldProvider: old.String(),
		NewProvider: provider.String(),
	})
	return nil
}

func (s *PlatformSettings) SMSActiveProvider() SMSProvider { return s.smsActiveProvider }

// SetPaymentActiveProvider, aktif ödeme sağlayıcısını değiştirir.
func (s *PlatformSettings) SetPaymentActiveProvider(provider PaymentProvider) error {
	if !IsValidPaymentProvider(provider) {
		return ErrInvalidPaymentProvider
	}
	if s.paymentActiveProvider == provider {
		return nil
	}
	old := s.paymentActiveProvider
	s.paymentActiveProvider = provider
	s.Record(PaymentProviderChangedEvent{
		BaseEvent:   shared.NewBaseEvent(EventPaymentProviderChanged, KeyPaymentActiveProvider.String()),
		OldProvider: old.String(),
		NewProvider: provider.String(),
	})
	return nil
}

func (s *PlatformSettings) PaymentActiveProvider() PaymentProvider { return s.paymentActiveProvider }
