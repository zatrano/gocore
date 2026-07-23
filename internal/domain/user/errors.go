package user

import "github.com/zatrano/gocore/internal/domain/shared"

// Domain-özel hatalar. Hepsi *shared.DomainError'dır; HTTP katmanı bunları
// RFC7807 Problem Details'e ve doğru durum koduna çevirir.
var (
	ErrInvalidID = shared.NewDomainError(
		shared.KindValidation, "user.invalid_id", "geçersiz kullanıcı kimliği")

	ErrEmailRequired = shared.NewDomainError(
		shared.KindValidation, "user.email_required", "e-posta adresi zorunludur")

	ErrInvalidEmail = shared.NewDomainError(
		shared.KindValidation, "user.invalid_email", "geçersiz e-posta adresi")

	ErrInvalidPhone = shared.NewDomainError(
		shared.KindValidation, "user.invalid_phone", "geçersiz telefon numarası")

	ErrNameRequired = shared.NewDomainError(
		shared.KindValidation, "user.name_required", "ad zorunludur")

	ErrInvalidPasswordHash = shared.NewDomainError(
		shared.KindValidation, "user.invalid_password_hash", "geçersiz şifre hash formatı")

	ErrNotFound = shared.NewDomainError(
		shared.KindNotFound, "user.not_found", "kullanıcı bulunamadı")

	ErrEmailAlreadyExists = shared.NewDomainError(
		shared.KindConflict, "user.email_exists", "bu e-posta adresi zaten kayıtlı")

	ErrAlreadyActive = shared.NewDomainError(
		shared.KindConflict, "user.already_active", "kullanıcı zaten aktif")

	ErrInactive = shared.NewDomainError(
		shared.KindForbidden, "user.inactive", "kullanıcı hesabı pasif")

	ErrAlreadyDeleted = shared.NewDomainError(
		shared.KindConflict, "user.already_deleted", "kullanıcı zaten silinmiş")

	ErrNotDeleted = shared.NewDomainError(
		shared.KindConflict, "user.not_deleted", "kullanıcı silinmiş değil")

	ErrLocaleRequired = shared.NewDomainError(
		shared.KindValidation, "user.locale_required", "dil tercihi zorunludur")

	ErrInvalidLocale = shared.NewDomainError(
		shared.KindValidation, "user.invalid_locale", "geçersiz dil kodu")

	ErrUnsupportedLocale = shared.NewDomainError(
		shared.KindValidation, "user.unsupported_locale", "desteklenmeyen dil")

	ErrEmailAlreadyVerified = shared.NewDomainError(
		shared.KindConflict, "user.email_already_verified", "e-posta adresi zaten doğrulanmış")

	ErrMFANotConfigured = shared.NewDomainError(
		shared.KindValidation, "user.mfa_not_configured", "iki adımlı doğrulama yapılandırılmamış")

	ErrMFAAlreadyEnabled = shared.NewDomainError(
		shared.KindConflict, "user.mfa_already_enabled", "iki adımlı doğrulama zaten etkin")

	ErrMFANotEnabled = shared.NewDomainError(
		shared.KindConflict, "user.mfa_not_enabled", "iki adımlı doğrulama etkin değil")

	ErrInvalidRole = shared.NewDomainError(
		shared.KindValidation, "user.invalid_role", "geçersiz rol")

	ErrSameRole = shared.NewDomainError(
		shared.KindConflict, "user.same_role", "kullanıcı zaten bu role sahip")

	ErrCannotDemoteLastAdmin = shared.NewDomainError(
		shared.KindConflict, "user.last_admin", "son admin kullanıcısının rolü değiştirilemez")
)
