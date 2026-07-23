package validation

import (
	"context"
	"fmt"

	"github.com/zatrano/gocore/pkg/i18n"
)

// InvalidField, sanitize aşamasında başarısız olan tek bir alanı tanımlar.
type InvalidField struct {
	Field string
	Tag   string // i18n: validation.<tag>
}

// InvalidFields, birden fazla alan sanitize hatası.
type InvalidFields []InvalidField

func (e InvalidFields) Error() string {
	return fmt.Sprintf("validation: %d alan geçersiz", len(e))
}

// FieldMap, GoUI formları için alan→mesaj haritası üretir.
func (e InvalidFields) FieldMap(ctx context.Context) map[string]string {
	out := make(map[string]string, len(e))
	for _, f := range e {
		key := "validation." + f.Tag
		fallback := "geçersiz değer"
		switch f.Tag {
		case "email":
			fallback = "geçerli bir e-posta adresi giriniz"
		case "phone":
			fallback = "geçerli bir telefon numarası giriniz (ör. +905551112233)"
		}
		out[f.Field] = i18n.T(ctx, key, fallback)
	}
	return out
}

// ProblemFields, API RFC7807 alan hatalarına çevirir.
func (e InvalidFields) ProblemFields(ctx context.Context) []struct {
	Field, Message string
} {
	out := make([]struct{ Field, Message string }, len(e))
	m := e.FieldMap(ctx)
	for i, f := range e {
		out[i].Field = f.Field
		out[i].Message = m[f.Field]
	}
	return out
}
