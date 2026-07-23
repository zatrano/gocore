package auth

import (
	"errors"

	domainauth "github.com/zatrano/gocore/internal/domain/auth"
)

// Domain hatalarının uygulama katmanı re-export'ları (handler uyumu).
var (
	ErrInvalidCredentials    = domainauth.ErrInvalidCredentials
	ErrTooManyAttempts       = domainauth.ErrTooManyAttempts
	ErrAccountInactive       = domainauth.ErrAccountInactive
	ErrMFARequired           = domainauth.ErrMFARequired
	ErrInvalidMFACode        = domainauth.ErrInvalidMFACode
	ErrInvalidToken          = domainauth.ErrInvalidToken
	ErrTokenReuse            = domainauth.ErrTokenReuse
	ErrWeakPassword          = domainauth.ErrWeakPassword
	ErrOAuthProviderNotFound = domainauth.ErrOAuthProviderNotFound
	ErrOAuthExchange         = domainauth.ErrOAuthExchange
)

// MFARequiredError, parola doğrulandıktan sonra istemciye yalnızca opak
// MFA challenge değerini taşır. Parola ikinci adıma aktarılmaz.
type MFARequiredError struct {
	Challenge string
}

func (e *MFARequiredError) Error() string { return ErrMFARequired.Error() }
func (e *MFARequiredError) Unwrap() error { return ErrMFARequired }

// MFAChallengeFrom, MFA gerekli hatasındaki opak challenge değerini döndürür.
func MFAChallengeFrom(err error) (string, bool) {
	var target *MFARequiredError
	if !errors.As(err, &target) || target.Challenge == "" {
		return "", false
	}
	return target.Challenge, true
}
