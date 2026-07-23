// Package render, HTTP yanıtlarını (başarı ve RFC7807 hata) tutarlı biçimde
// yazmak için yardımcılar sağlar. Hem handler'lar hem router bunu kullanır.
// Hata başlık/detay ve doğrulama mesajları, isteğin çözümlenmiş diline göre
// (i18n) yerelleştirilir.
package render

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/problem"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/validation"
)

// JSON, veriyi verilen durum koduyla JSON olarak yazar.
func JSON(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}

// MessageText, yerelleştirilmiş mesaj metnini döner (gömülü yanıt gövdeleri için).
func MessageText(c fiber.Ctx, key, fallback string, args ...any) string {
	return i18n.T(c.Context(), key, fallback, args...)
}

// Message, yerelleştirilmiş başarı mesajı döner: {"message": "..."}.
func Message(c fiber.Ctx, status int, key, fallback string, args ...any) error {
	return JSON(c, status, fiber.Map{
		"message": i18n.T(c.Context(), key, fallback, args...),
	})
}

// JSONWithMessage, veri + yerelleştirilmiş başarı mesajı döner:
// {"message": "...", "data": ...}.
func JSONWithMessage(c fiber.Ctx, status int, key, fallback string, data any, args ...any) error {
	return JSON(c, status, fiber.Map{
		"message": i18n.T(c.Context(), key, fallback, args...),
		"data":    data,
	})
}

// ProblemLocalized, RFC7807 hatasını isteğin diline çevirip yazar.
func ProblemLocalized(c fiber.Ctx, status int, code, titleKey, titleFallback, detailKey, detailFallback string, args ...any) error {
	ctx := c.Context()
	p := problem.New(status, code,
		i18n.T(ctx, titleKey, titleFallback),
		i18n.T(ctx, detailKey, detailFallback, args...))
	return Problem(c, p)
}

// Problem, RFC7807 problem+json olarak hata yazar.
func Problem(c fiber.Ctx, p *problem.Problem) error {
	p.Instance = c.Path()
	return c.Status(p.Status).JSON(p, problem.ContentType)
}

// Error, herhangi bir hatayı uygun RFC7807 problemine çevirip yazar. İç hata
// detayları istemciye sızdırılmaz (bilgi ifşasına karşı). Mesajlar isteğin
// diline yerelleştirilir.
func Error(c fiber.Ctx, err error) error {
	ctx := c.Context()

	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		p := problem.WithValidation(toFieldErrors(c, verrs))
		p.Title = i18n.T(ctx, "title.validation", p.Title)
		p.Detail = i18n.T(ctx, "validation.detail", p.Detail)
		return Problem(c, p)
	}

	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		fields := make([]problem.FieldError, 0, len(inv))
		for name, msg := range inv.FieldMap(ctx) {
			fields = append(fields, problem.FieldError{Field: name, Message: msg})
		}
		p := problem.WithValidation(fields)
		p.Title = i18n.T(ctx, "title.validation", p.Title)
		p.Detail = i18n.T(ctx, "validation.detail", p.Detail)
		return Problem(c, p)
	}

	if de, ok := shared.AsDomainError(err); ok {
		p := problem.FromDomain(de)
		p.Title = i18n.T(ctx, problem.TitleKey(de.Kind), p.Title)
		p.Detail = i18n.T(ctx, de.Code, p.Detail)
		return Problem(c, p)
	}

	p := problem.New(500, "internal_error",
		i18n.T(ctx, "title.internal", "Sunucu hatası"),
		i18n.T(ctx, "internal_error", "Beklenmeyen bir hata oluştu"))
	return Problem(c, p)
}

func toFieldErrors(c fiber.Ctx, verrs validator.ValidationErrors) []problem.FieldError {
	out := make([]problem.FieldError, 0, len(verrs))
	for _, fe := range verrs {
		out = append(out, problem.FieldError{Field: fe.Field(), Message: validation.FieldMessage(c.Context(), fe)})
	}
	return out
}
