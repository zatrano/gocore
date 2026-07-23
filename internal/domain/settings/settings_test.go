package settings_test

import (
	"errors"
	"testing"

	"github.com/zatrano/gocore/internal/domain/settings"
)

func TestPlatformSettings_SetSMSActiveProvider(t *testing.T) {
	s := settings.DefaultPlatformSettings()
	if err := s.SetSMSActiveProvider(settings.ProviderNetgsm); err != nil {
		t.Fatalf("SetSMSActiveProvider: %v", err)
	}
	if s.SMSActiveProvider() != settings.ProviderNetgsm {
		t.Fatalf("sms provider = %q", s.SMSActiveProvider())
	}
}

func TestPlatformSettings_SetPaymentActiveProvider(t *testing.T) {
	s := settings.DefaultPlatformSettings()
	if err := s.SetPaymentActiveProvider(settings.ProviderIyzico); err != nil {
		t.Fatalf("SetPaymentActiveProvider: %v", err)
	}
	if s.PaymentActiveProvider() != settings.ProviderIyzico {
		t.Fatalf("payment provider = %q", s.PaymentActiveProvider())
	}
}

func TestRehydratePlatformSettings(t *testing.T) {
	s := settings.RehydratePlatformSettings("netgsm", "moka")
	if s.SMSActiveProvider() != settings.ProviderNetgsm {
		t.Fatalf("sms = %q", s.SMSActiveProvider())
	}
	if s.PaymentActiveProvider() != settings.ProviderMoka {
		t.Fatalf("payment = %q", s.PaymentActiveProvider())
	}
}

func TestDefaultPlatformSettings_PaymentProvider(t *testing.T) {
	s := settings.DefaultPlatformSettings()
	if s.PaymentActiveProvider() != settings.ProviderIyzico {
		t.Fatalf("default payment = %q", s.PaymentActiveProvider())
	}
}

func TestParseSMSProvider_Invalid(t *testing.T) {
	_, err := settings.ParseSMSProvider("twilio")
	if !errors.Is(err, settings.ErrInvalidSMSProvider) {
		t.Fatalf("expected ErrInvalidSMSProvider, got %v", err)
	}
}

func TestParsePaymentProvider_Invalid(t *testing.T) {
	_, err := settings.ParsePaymentProvider("paypal")
	if !errors.Is(err, settings.ErrInvalidPaymentProvider) {
		t.Fatalf("expected ErrInvalidPaymentProvider, got %v", err)
	}
}
