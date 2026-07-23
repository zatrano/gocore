package security

import (
	"crypto/subtle"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP, auth.TOTP portunun RFC 6238 (pquerna/otp) tabanlı implementasyonudur.
type TOTP struct {
	issuer string
}

// NewTOTP, verilen issuer (uygulama adı) ile TOTP üreticisini kurar.
func NewTOTP(issuer string) *TOTP { return &TOTP{issuer: issuer} }

// Generate, yeni bir paylaşılan sır ve otpauth:// URI üretir. URI, authenticator
// uygulamasında QR kod olarak taranır.
func (t *TOTP) Generate(accountName string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      t.issuer,
		AccountName: accountName,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// Validate, verilen sır için kodun eşleştiği RFC 6238 zaman adımını döner.
// Küçük saat kaymalarına toleranslıdır (±1 pencere).
func (t *TOTP) Validate(secret, code string) (int64, bool) {
	const period int64 = 30
	current := time.Now().UTC().Unix() / period
	opts := totp.ValidateOpts{
		Period:    uint(period),
		Skew:      0,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
	for _, step := range []int64{current, current - 1, current + 1} {
		candidate, err := totp.GenerateCodeCustom(secret, time.Unix(step*period, 0).UTC(), opts)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}
