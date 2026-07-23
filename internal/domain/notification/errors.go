package notification

import "github.com/zatrano/gocore/internal/domain/shared"

// Domain-özel hatalar. Hepsi *shared.DomainError'dır; HTTP katmanı bunları
// RFC7807 Problem Details'e ve doğru durum koduna çevirir.
var (
	ErrInvalidID = shared.NewDomainError(
		shared.KindValidation, "notification.invalid_id", "geçersiz bildirim kimliği")

	ErrRecipientRequired = shared.NewDomainError(
		shared.KindValidation, "notification.recipient_required", "alıcı zorunludur")

	ErrTitleRequired = shared.NewDomainError(
		shared.KindValidation, "notification.title_required", "başlık zorunludur")

	ErrContentRequired = shared.NewDomainError(
		shared.KindValidation, "notification.content_required", "içerik zorunludur")

	ErrNotFound = shared.NewDomainError(
		shared.KindNotFound, "notification.not_found", "bildirim bulunamadı")
)
