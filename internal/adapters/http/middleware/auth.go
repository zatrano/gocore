package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/problem"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/rbac"
)

// TokenVerifier, access token doğrulaması yapan porttur. SessionManager bunu
// uygular ve iptal (logout/toplu iptal) kontrolünü de içerir.
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (auth.Claims, error)
}

// Authenticator, Bearer token doğrulaması yapar. Geçersiz/eksik token'da 401
// döner; geçerliyse claim'leri Locals ve context'e yazar.
func Authenticator(verifier TokenVerifier) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		if header == "" {
			return writeProblem(c, problem.New(401, "auth.missing_token", "Kimlik doğrulama gerekli", "Authorization başlığı eksik"))
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return writeProblem(c, problem.New(401, "auth.invalid_scheme", "Kimlik doğrulama gerekli", "Bearer şeması bekleniyor"))
		}

		claims, err := verifier.Verify(c.Context(), parts[1])
		if err != nil {
			return writeProblem(c, problem.New(401, "auth.invalid_token", "Kimlik doğrulama gerekli", "Token geçersiz veya süresi dolmuş"))
		}

		c.Locals(adapters.LocalUserID, claims.UserID)
		c.Locals(adapters.LocalRole, claims.Role)
		c.Locals(adapters.LocalEmail, claims.Email)

		// user_id'yi loglara taşımak için context'e ekle.
		c.SetContext(logger.WithUserID(c.Context(), claims.UserID))
		return c.Next()
	}
}

// RequirePermission, çağıranın belirtilen izinlerden en az birine sahip olmasını
// zorunlu kılar. Yetki çözümlemesi dinamiktir (rbac.Checker → DB destekli
// resolver). Authenticator'dan sonra kullanılmalıdır. Çözümleme hatasında
// güvenli tarafta kalınır (403).
func RequirePermission(checker rbac.Checker, perms ...rbac.Permission) fiber.Handler {
	return func(c fiber.Ctx) error {
		role, _ := c.Locals(adapters.LocalRole).(string)
		for _, p := range perms {
			ok, err := checker.Allows(c.Context(), role, p)
			if err == nil && ok {
				return c.Next()
			}
		}
		return writeProblem(c, problem.New(403, "auth.forbidden", "Erişim reddedildi", "Bu işlem için yetkiniz yok"))
	}
}

// writeProblem, middleware içinden RFC7807 hata yazar. Başlık ve detay, isteğin
// çözümlenmiş diline (i18n) yerelleştirilir.
func writeProblem(c fiber.Ctx, p *problem.Problem) error {
	p.Instance = c.Path()
	p.Detail = i18n.T(c.Context(), p.Code, p.Detail)
	p.Title = i18n.T(c.Context(), titleKeyForStatus(p.Status), p.Title)
	return c.Status(p.Status).JSON(p, problem.ContentType)
}

// titleKeyForStatus, HTTP durum koduna karşılık gelen i18n başlık anahtarını döner.
func titleKeyForStatus(status int) string {
	switch status {
	case 403:
		return "title.forbidden"
	case 429:
		return "title.rate_limited"
	default:
		return "title.unauthorized"
	}
}
