package moka

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// CheckKey, Moka kimlik doğrulama anahtarını üretir.
// SHA256(DealerCode + "MK" + Username + "PD" + Password)
func CheckKey(dealerCode, username, password string) string {
	raw := dealerCode + "MK" + username + "PD" + password
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// HashResult, CodeForHash ile ödeme sonucu hash'idir.
func HashResult(codeForHash, suffix string) string {
	code := strings.ToUpper(strings.TrimSpace(codeForHash))
	sum := sha256.Sum256([]byte(code + suffix))
	return hex.EncodeToString(sum[:])
}

// VerifyHashValue, callback hashValue değerini doğrular.
// Başarı: SHA256(UPPER(CodeForHash)+"T"), başarısızlık: ...+"F"
func VerifyHashValue(hashValue, codeForHash string) (success bool, valid bool) {
	hashValue = strings.ToLower(strings.TrimSpace(hashValue))
	if hashValue == "" {
		return false, false
	}
	successHash := strings.ToLower(HashResult(codeForHash, "T"))
	failHash := strings.ToLower(HashResult(codeForHash, "F"))
	switch hashValue {
	case successHash:
		return true, true
	case failHash:
		return false, true
	default:
		return false, false
	}
}
