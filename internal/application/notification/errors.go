package notification

import "github.com/zatrano/gocore/internal/domain/shared"

// Merkezi bildirim sisteminin doğrulama hataları. Hepsi *shared.DomainError'dır;
// HTTP katmanı bunları RFC7807'ye ve doğru status koduna çevirir.
var (
	ErrUnsupportedChannel = shared.NewDomainError(
		shared.KindValidation, "notification.unsupported_channel", "desteklenmeyen bildirim türü")

	ErrRecipientRequired = shared.NewDomainError(
		shared.KindValidation, "notification.recipient_required", "alıcı zorunludur")

	ErrRecipientNotFound = shared.NewDomainError(
		shared.KindValidation, "notification.recipient_not_found",
		"bu e-posta ile kayıtlı aktif kullanıcı bulunamadı")

	ErrPhoneRequired = shared.NewDomainError(
		shared.KindValidation, "notification.phone_required", "telefon numarası zorunludur")

	ErrEmailRequired = shared.NewDomainError(
		shared.KindValidation, "notification.email_required", "e-posta adresi zorunludur")

	ErrTitleRequired = shared.NewDomainError(
		shared.KindValidation, "notification.title_required", "başlık zorunludur")

	ErrContentRequired = shared.NewDomainError(
		shared.KindValidation, "notification.content_required", "içerik zorunludur")

	ErrAudienceUnsupported = shared.NewDomainError(
		shared.KindValidation, "notification.audience_unsupported",
		"tüm kullanıcılara gönderim bu kanal için desteklenmiyor")

	ErrUserDirectoryRequired = shared.NewDomainError(
		shared.KindInternal, "notification.user_directory_required",
		"kullanıcı dizini yapılandırılmamış")
)
