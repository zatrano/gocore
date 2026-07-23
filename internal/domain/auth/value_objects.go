package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/google/uuid"
)

// TokenPurpose, tek kullanımlık token'ın amacını belirtir.
type TokenPurpose string

const (
	PurposeEmailVerify   TokenPurpose = "email_verify"
	PurposePasswordReset TokenPurpose = "password_reset"
)

func (p TokenPurpose) String() string { return string(p) }

// TokenID, tek kullanımlık token kaydının kimliğidir.
type TokenID struct {
	value uuid.UUID
}

// NewTokenID, yeni bir token kimliği üretir.
func NewTokenID() TokenID {
	v, err := uuid.NewV7()
	if err != nil {
		v = uuid.New()
	}
	return TokenID{value: v}
}

// ParseTokenID, string UUID'yi TokenID'ye çevirir.
func ParseTokenID(s string) (TokenID, error) {
	v, err := uuid.Parse(s)
	if err != nil {
		return TokenID{}, ErrInvalidToken
	}
	return TokenID{value: v}, nil
}

func TokenIDFromUUID(v uuid.UUID) TokenID { return TokenID{value: v} }

func (id TokenID) UUID() uuid.UUID { return id.value }
func (id TokenID) String() string  { return id.value.String() }
func (id TokenID) IsZero() bool    { return id.value == uuid.Nil }

// GenerateRawToken, URL-safe kriptografik rastgele token üretir (ham değer).
func GenerateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken, ham token'ın SHA-256 özetini hex olarak döner.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

const MinPasswordLength = 8

// ValidatePasswordLength, şifre uzunluk politikasını doğrular.
func ValidatePasswordLength(plain string) error {
	if len(plain) < MinPasswordLength {
		return ErrWeakPassword
	}
	return nil
}
