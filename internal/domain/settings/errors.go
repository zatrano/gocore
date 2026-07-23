package settings

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	ErrInvalidSMSProvider = shared.NewDomainError(
		shared.KindValidation, "settings.invalid_sms_provider", "geçersiz SMS sağlayıcısı")
	ErrInvalidPaymentProvider = shared.NewDomainError(
		shared.KindValidation, "settings.invalid_payment_provider", "geçersiz ödeme sağlayıcısı")
)
