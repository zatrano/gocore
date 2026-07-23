package contact

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	ErrInvalidID = shared.NewDomainError(
		shared.KindValidation, "contact.invalid_id", "geçersiz iletişim mesajı kimliği")
	ErrInvalidEmail = shared.NewDomainError(
		shared.KindValidation, "contact.invalid_email", "geçersiz e-posta")
	ErrInvalidName = shared.NewDomainError(
		shared.KindValidation, "contact.invalid_name", "ad 2-100 karakter olmalıdır")
	ErrInvalidMessage = shared.NewDomainError(
		shared.KindValidation, "contact.invalid_message", "mesaj 5-2000 karakter olmalıdır")
	ErrNotFound = shared.NewDomainError(
		shared.KindNotFound, "contact.not_found", "iletişim mesajı bulunamadı")
)
