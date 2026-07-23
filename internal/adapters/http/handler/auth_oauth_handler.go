package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
)

// OAuthStart, GET /auth/oauth/:provider — kullanıcıyı sağlayıcının yetkilendirme
// sayfasına yönlendirir. CSRF state'i tek kullanımlık cache kaydıdır.
func (h *AuthHandler) OAuthStart(c fiber.Ctx) error {
	provider := c.Params("provider")
	state, err := adapters.IssueOAuthState(
		c.Context(),
		h.cache,
		adapters.OAuthStateAPIPrefix,
		adapters.OAuthStatePayload{Provider: provider},
	)
	if err != nil {
		return render.Error(c, err)
	}
	url, err := h.auth.OAuthAuthCodeURL(provider, state)
	if err != nil {
		return render.Error(c, err)
	}
	return c.Redirect().To(url)
}

// OAuthCallback, GET /auth/oauth/:provider/callback — sağlayıcıdan dönen kodu
// işler, kullanıcıyı bul/oluştur ve token çifti döner.
func (h *AuthHandler) OAuthCallback(c fiber.Ctx) error {
	provider := c.Params("provider")
	if _, ok := adapters.ConsumeOAuthState(
		c.Context(),
		h.cache,
		adapters.OAuthStateAPIPrefix,
		c.Query("state"),
		provider,
	); !ok {
		return render.ProblemLocalized(c, fiber.StatusUnauthorized, "auth.oauth_state_mismatch",
			"title.unauthorized", "Kimlik doğrulama gerekli",
			"auth.oauth_state_mismatch", "OAuth state doğrulaması başarısız")
	}
	code := c.Query("code")
	if code == "" {
		return render.ProblemLocalized(c, fiber.StatusUnauthorized, "auth.oauth_exchange_failed",
			"title.unauthorized", "Kimlik doğrulama gerekli",
			"auth.oauth_exchange_failed", "OAuth doğrulaması başarısız")
	}

	tokens, err := h.auth.OAuthCallback(c.Context(), provider, code)
	if err != nil {
		return render.Error(c, err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return render.JSON(c, fiber.StatusOK, tokens)
}
