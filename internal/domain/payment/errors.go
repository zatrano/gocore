package payment

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	ErrProviderNotConfigured = shared.NewDomainError(
		shared.KindValidation, "payment.provider_not_configured", "ödeme sağlayıcısı yapılandırılmamış")
	ErrProviderNotActive = shared.NewDomainError(
		shared.KindValidation, "payment.provider_not_active", "aktif ödeme sağlayıcısı bu işlem için uygun değil")
	ErrPaymentNotFound = shared.NewDomainError(
		shared.KindNotFound, "payment.not_found", "ödeme kaydı bulunamadı")
	ErrInvalidCallback = shared.NewDomainError(
		shared.KindValidation, "payment.invalid_callback", "geçersiz 3DS callback")
	ErrInvalidBin = shared.NewDomainError(
		shared.KindValidation, "payment.invalid_bin", "geçersiz BIN numarası")
	ErrBinPriceRequired = shared.NewDomainError(
		shared.KindValidation, "payment.bin_price_required", "iyzico BIN sorgusu için tutar zorunludur")
	ErrThreeDSFailed = shared.NewDomainError(
		shared.KindValidation, "payment.threeds_failed", "3DS doğrulaması başarısız")
	ErrWebhookInvalidSignature = shared.NewDomainError(
		shared.KindUnauthorized, "payment.webhook_invalid_signature", "geçersiz iyzico webhook imzası")
	ErrWebhookSignatureRequired = shared.NewDomainError(
		shared.KindUnauthorized, "payment.webhook_signature_required", "X-IYZ-SIGNATURE-V3 başlığı zorunludur")
)
