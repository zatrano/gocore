// Package shared, tüm bounded context'lerin paylaştığı domain yapı taşlarını
// içerir: domain hataları, event altyapısı ve ortak value object'ler.
package shared

import "errors"

// DomainError, iş kuralı ihlallerini temsil eden hata tipidir. Altyapı
// hatalarından (DB, ağ vb.) ayrıştırılır; HTTP katmanı bunları anlamlı
// durum kodlarına çevirir.
type DomainError struct {
	// Code, makine tarafından okunabilir hata kodu (ör. "user.not_found").
	Code string
	// Message, kullanıcıya gösterilebilir açıklama.
	Message string
	// Kind, hatanın kategorisi; HTTP durum kodu eşlemesi için kullanılır.
	Kind ErrorKind
	// cause, sarmalanan alt hata (opsiyonel).
	cause error
}

// ErrorKind, domain hatasının kategorisi.
type ErrorKind uint8

const (
	// KindValidation, girdi/iş kuralı doğrulama hatası (400).
	KindValidation ErrorKind = iota
	// KindNotFound, kayıt bulunamadı (404).
	KindNotFound
	// KindConflict, benzersizlik/durum çakışması (409).
	KindConflict
	// KindUnauthorized, kimlik doğrulama gerekli (401).
	KindUnauthorized
	// KindForbidden, yetki yetersiz (403).
	KindForbidden
	// KindRateLimited, hız sınırı aşıldı (429).
	KindRateLimited
	// KindInternal, beklenmeyen iç hata (500).
	KindInternal
)

func (e *DomainError) Error() string { return e.Code + ": " + e.Message }
func (e *DomainError) Unwrap() error { return e.cause }

// WithCause, hataya bir alt neden iliştirir (immutable kopya döner).
func (e *DomainError) WithCause(err error) *DomainError {
	clone := *e
	clone.cause = err
	return &clone
}

// NewDomainError, yeni bir domain hatası oluşturur.
func NewDomainError(kind ErrorKind, code, message string) *DomainError {
	return &DomainError{Kind: kind, Code: code, Message: message}
}

// AsDomainError, verilen hatayı *DomainError olarak çözümlemeye çalışır.
func AsDomainError(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
