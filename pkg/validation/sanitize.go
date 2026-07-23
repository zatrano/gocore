package validation

import (
	"errors"
	"net/mail"
	"strings"
)

// E164Pattern, OpenAPI ve DB CHECK ile uyumlu E.164 telefon deseni.
const E164Pattern = `^\+[1-9][0-9]{7,14}$`

var (
	// ErrInvalidEmail, geçersiz e-posta biçimi.
	ErrInvalidEmail = errors.New("validation: invalid email")
	// ErrInvalidPhone, geçersiz telefon biçimi.
	ErrInvalidPhone = errors.New("validation: invalid phone")
)

// NormalizeEmail, e-postayı trim + lowercase yapar ve doğrular. Boş girdi ("", nil) döner.
func NormalizeEmail(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "", nil
	}
	if len(normalized) > 254 {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// NormalizePhone, telefonu E.164'e çevirir. Boş girdi ("", nil) döner.
func NormalizePhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	var b strings.Builder
	b.Grow(len(raw) + 1)
	for i, r := range raw {
		switch {
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '(' || r == ')':
			continue
		default:
			return "", ErrInvalidPhone
		}
	}
	s := b.String()
	if s == "" {
		return "", ErrInvalidPhone
	}
	switch {
	case strings.HasPrefix(s, "+"):
	case strings.HasPrefix(s, "00"):
		s = "+" + s[2:]
	case strings.HasPrefix(s, "0") && len(s) == 11:
		s = "+90" + s[1:]
	case strings.HasPrefix(s, "90") && len(s) == 12:
		s = "+" + s
	case len(s) == 10 && s[0] == '5':
		s = "+90" + s
	default:
		if !strings.HasPrefix(s, "+") {
			return "", ErrInvalidPhone
		}
	}
	digits := strings.TrimPrefix(s, "+")
	if len(digits) < 8 || len(digits) > 15 {
		return "", ErrInvalidPhone
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", ErrInvalidPhone
		}
	}
	return s, nil
}
