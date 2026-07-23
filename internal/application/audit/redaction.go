package audit

import (
	"strings"
)

var sensitiveKeyFragments = []string{
	"password", "secret", "token", "authorization", "cookie", "apikey", "api_key",
	"card", "pan", "cvv", "cvc", "expiry", "conversation", "signature", "payload",
	"html", "body", "content", "message", "recipient",
}

// RedactMetadata, denetim metadata'sından hassas alanları temizler.
func RedactMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		lk := strings.ToLower(k)
		if isSensitiveKey(lk) {
			out[k] = "[redacted]"
			continue
		}
		switch t := v.(type) {
		case map[string]any:
			out[k] = RedactMetadata(t)
		case []any:
			out[k] = redactSlice(t)
		case string:
			out[k] = redactStringValue(lk, t)
		default:
			out[k] = v
		}
	}
	return out
}

func redactSlice(in []any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		switch t := v.(type) {
		case map[string]any:
			out[i] = RedactMetadata(t)
		case []any:
			out[i] = redactSlice(t)
		default:
			out[i] = t
		}
	}
	return out
}

func isSensitiveKey(k string) bool {
	for _, frag := range sensitiveKeyFragments {
		if strings.Contains(k, frag) {
			// email/phone gibi alanlar allowlist ile geçsin
			if k == "email" || k == "old_email" || k == "new_email" ||
				k == "phone" || k == "old_phone" || k == "new_phone" ||
				k == "name" || k == "old_name" || k == "new_name" {
				return false
			}
			if frag == "content" && (k == "content_type" || k == "content_hash") {
				return false
			}
			return true
		}
	}
	return false
}

func redactStringValue(key, v string) string {
	switch key {
	case "email", "old_email", "new_email", "actor_email":
		return MaskEmail(v)
	case "phone", "old_phone", "new_phone":
		return MaskPhone(v)
	default:
		return v
	}
}

// MaskEmail, e-postayı a***@example.com biçiminde maskeler.
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return "***"
	}
	local, domain := email[:at], email[at+1:]
	if len(local) == 1 {
		return local + "***@" + domain
	}
	return string(local[0]) + "***@" + domain
}

// MaskPhone, telefonun yalnızca son 4 hanesini bırakır.
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	if len(phone) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}
