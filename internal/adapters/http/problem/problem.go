// Package problem, RFC 7807 (Problem Details for HTTP APIs) uyumlu hata
// gövdeleri üretir. Tüm API hataları bu formatta döner; böylece istemciler
// tutarlı ve makine-okunabilir hata yanıtları alır.
package problem

import (
	"github.com/zatrano/gocore/internal/domain/shared"
)

// ContentType, RFC 7807 media type'ı.
const ContentType = "application/problem+json"

// Problem, RFC 7807 problem detayı gövdesidir. Immutable kabul edilir.
type Problem struct {
	// Type, hatayı tanımlayan URI (dokümantasyon bağlantısı olabilir).
	Type string `json:"type"`
	// Title, hatanın kısa, insan-okunabilir özeti.
	Title string `json:"title"`
	// Status, HTTP durum kodu.
	Status int `json:"status"`
	// Detail, bu olaya özgü açıklama.
	Detail string `json:"detail,omitempty"`
	// Instance, hatanın oluştuğu istek yolu.
	Instance string `json:"instance,omitempty"`
	// Code, uygulamaya özgü makine-okunabilir hata kodu.
	Code string `json:"code,omitempty"`
	// Errors, alan-bazlı doğrulama hataları (opsiyonel).
	Errors []FieldError `json:"errors,omitempty"`
}

// FieldError, tek bir alanın doğrulama hatasıdır.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// New, temel bir Problem üretir.
func New(status int, code, title, detail string) *Problem {
	return &Problem{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: detail,
		Code:   code,
	}
}

// FromDomain, bir domain hatasını uygun HTTP durum koduyla Problem'e çevirir.
func FromDomain(err *shared.DomainError) *Problem {
	status := statusFromKind(err.Kind)
	return &Problem{
		Type:   "about:blank",
		Title:  titleFromKind(err.Kind),
		Status: status,
		Detail: err.Message,
		Code:   err.Code,
	}
}

// WithValidation, alan hatalarını ekleyerek doğrulama problemi üretir.
func WithValidation(fieldErrors []FieldError) *Problem {
	return &Problem{
		Type:   "about:blank",
		Title:  "Doğrulama hatası",
		Status: 422,
		Detail: "Bir veya daha fazla alan geçersiz",
		Code:   "validation_failed",
		Errors: fieldErrors,
	}
}

func statusFromKind(k shared.ErrorKind) int {
	switch k {
	case shared.KindValidation:
		return 422
	case shared.KindNotFound:
		return 404
	case shared.KindConflict:
		return 409
	case shared.KindUnauthorized:
		return 401
	case shared.KindForbidden:
		return 403
	case shared.KindRateLimited:
		return 429
	default:
		return 500
	}
}

// TitleKey, bir hata türü için i18n çeviri anahtarını döner (ör. "title.not_found").
// render katmanı, başlığı istemcinin diline çevirmek için bunu kullanır.
func TitleKey(k shared.ErrorKind) string {
	switch k {
	case shared.KindValidation:
		return "title.validation"
	case shared.KindNotFound:
		return "title.not_found"
	case shared.KindConflict:
		return "title.conflict"
	case shared.KindUnauthorized:
		return "title.unauthorized"
	case shared.KindForbidden:
		return "title.forbidden"
	case shared.KindRateLimited:
		return "title.rate_limited"
	default:
		return "title.internal"
	}
}

func titleFromKind(k shared.ErrorKind) string {
	switch k {
	case shared.KindValidation:
		return "Geçersiz istek"
	case shared.KindNotFound:
		return "Bulunamadı"
	case shared.KindConflict:
		return "Çakışma"
	case shared.KindUnauthorized:
		return "Kimlik doğrulama gerekli"
	case shared.KindForbidden:
		return "Erişim reddedildi"
	case shared.KindRateLimited:
		return "Çok fazla istek"
	default:
		return "Sunucu hatası"
	}
}
