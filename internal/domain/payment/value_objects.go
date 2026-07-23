package payment

import "strings"

// PaymentStatus, kullanıcıya yönelik ödeme sonucudur.
type PaymentStatus string

const (
	StatusPending PaymentStatus = "pending"
	StatusSuccess PaymentStatus = "success"
	StatusFailed  PaymentStatus = "failed"
)

// PaymentStage, 3DS akışındaki teknik aşamadır.
type PaymentStage string

const (
	StageInitialized PaymentStage = "initialized"
	StageCallbackOK  PaymentStage = "callback_ok"
	StageCompleted   PaymentStage = "completed"
	StageFailed      PaymentStage = "failed"
)

// StatusLabel, durum için Türkçe etiket döner.
func StatusLabel(status PaymentStatus) string {
	switch status {
	case StatusSuccess:
		return "Başarılı"
	case StatusFailed:
		return "Başarısız"
	default:
		return "Beklemede"
	}
}

// CardMask, kart numarasından BIN ve son 4 haneyi çıkarır.
func CardMask(cardNumber string) (bin, last4 string) {
	var digits strings.Builder
	for _, r := range cardNumber {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	s := digits.String()
	if len(s) >= 6 {
		bin = s[:6]
	}
	if len(s) >= 4 {
		last4 = s[len(s)-4:]
	}
	return bin, last4
}

// CardDisplay, maskeli kart gösterimi üretir (yalnızca ilk 2 BIN hanesi görünür).
func CardDisplay(bin, last4 string) string {
	if last4 == "" {
		return ""
	}
	prefix := "****"
	if len(bin) >= 2 {
		prefix = bin[:2] + "****"
	}
	return prefix + last4
}

// MaskCardHolder, kart sahibi adını API/GoUI için maskeler.
func MaskCardHolder(holder string) string {
	holder = strings.TrimSpace(holder)
	if holder == "" {
		return ""
	}
	parts := strings.Fields(holder)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		masked := string(runes[0])
		if len(runes) > 1 {
			masked += strings.Repeat("*", len(runes)-1)
		}
		out = append(out, masked)
	}
	return strings.Join(out, " ")
}
