// Package validation, doğrulama hata mesajlarının i18n çevirisini sağlar.
// HTTP JSON ve GoUI adapter'ları ortak kullanır.
package validation

import (
	"context"

	"github.com/go-playground/validator/v10"

	"github.com/zatrano/gocore/pkg/i18n"
)

// FieldMessage, tek bir alan doğrulama hatasını yerelleştirir.
func FieldMessage(ctx context.Context, fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return i18n.T(ctx, "validation.required", "bu alan zorunludur")
	case "email":
		return i18n.T(ctx, "validation.email", "geçerli bir e-posta adresi giriniz")
	case "phone", "phone_required":
		return i18n.T(ctx, "validation.phone", "geçerli bir telefon numarası giriniz (ör. +905551112233)")
	case "min":
		return i18n.T(ctx, "validation.min", "en az "+fe.Param()+" karakter/değer olmalıdır", fe.Param())
	case "max":
		return i18n.T(ctx, "validation.max", "en fazla "+fe.Param()+" karakter/değer olmalıdır", fe.Param())
	case "oneof":
		return i18n.T(ctx, "validation.oneof", "izin verilen değerlerden biri olmalıdır: "+fe.Param(), fe.Param())
	default:
		return i18n.T(ctx, "validation.default", "geçersiz değer")
	}
}
