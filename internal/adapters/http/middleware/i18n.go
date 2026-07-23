package middleware

import (
	"github.com/gofiber/fiber/v3"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/pkg/i18n"
)

// Geriye dönük uyumluluk — yeni kod adapters/shared sabitlerini kullanmalıdır.
const (
	LocalLocale = adapters.LocalLocale
	LangCookie  = adapters.LangCookie
)

// Locale, isteğin dilini çözümler ve context'e iliştirir. Öncelik sırası:
//  1. ?lang=xx sorgu parametresi
//  2. "lang" cookie'si (kullanıcının önceki seçimi)
//  3. Accept-Language başlığı (q-değerlerine göre)
//  4. çevirmenin varsayılan dili
//
// Çözümlenen dil hem Locals'a hem Content-Language yanıt başlığına yazılır ve
// alt katmanların (render, handler) erişebilmesi için Go context'ine eklenir.
// Diğer middleware ve route'lardan ÖNCE çalışmalıdır.
func Locale(tr *i18n.Translator) fiber.Handler {
	return func(c fiber.Ctx) error {
		explicit := c.Query("lang")
		if explicit == "" {
			explicit = c.Cookies(LangCookie)
		}
		loc := tr.Resolve(explicit, c.Get(fiber.HeaderAcceptLanguage))
		c.Locals(LocalLocale, string(loc))
		c.Set(fiber.HeaderContentLanguage, string(loc))
		c.SetContext(i18n.NewContext(c.Context(), tr, loc))
		return c.Next()
	}
}
