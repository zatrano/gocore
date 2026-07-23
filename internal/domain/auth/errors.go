package auth

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	ErrInvalidCredentials = shared.NewDomainError(
		shared.KindUnauthorized, "auth.invalid_credentials", "e-posta veya şifre hatalı")

	ErrTooManyAttempts = shared.NewDomainError(
		shared.KindRateLimited, "auth.too_many_attempts", "çok fazla başarısız deneme, lütfen sonra tekrar deneyin")

	ErrAccountInactive = shared.NewDomainError(
		shared.KindForbidden, "auth.account_inactive", "hesap aktif değil")

	ErrMFARequired = shared.NewDomainError(
		shared.KindUnauthorized, "auth.mfa_required", "iki adımlı doğrulama kodu gerekli")

	ErrInvalidMFACode = shared.NewDomainError(
		shared.KindUnauthorized, "auth.invalid_mfa_code", "iki adımlı doğrulama kodu hatalı")

	ErrInvalidToken = shared.NewDomainError(
		shared.KindUnauthorized, "auth.invalid_token", "token geçersiz veya süresi dolmuş")

	ErrTokenReuse = shared.NewDomainError(
		shared.KindUnauthorized, "auth.token_reuse", "oturum güvenliği ihlali, lütfen tekrar giriş yapın")

	ErrWeakPassword = shared.NewDomainError(
		shared.KindValidation, "auth.weak_password", "şifre en az 8 karakter olmalıdır")

	ErrOAuthProviderNotFound = shared.NewDomainError(
		shared.KindValidation, "auth.oauth_provider_not_found", "OAuth sağlayıcısı bulunamadı")

	ErrOAuthExchange = shared.NewDomainError(
		shared.KindUnauthorized, "auth.oauth_exchange_failed", "OAuth doğrulaması başarısız")
)
