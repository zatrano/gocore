package shared

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
)

// VerifyTurnstile, public formlardan gelen Turnstile token'ını doğrular.
// Doğrulayıcı devre dışıysa nil döner.
func VerifyTurnstile(c fiber.Ctx, v turnstile.Verifier, token string) error {
	if v == nil || !v.Enabled() {
		return nil
	}
	return v.Verify(c.Context(), token, c.IP())
}
