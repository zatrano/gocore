package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	"github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/pkg/rbac"
)

// AuthHandler, tüm kimlik doğrulama uç noktalarını sağlar: login, refresh,
// logout, şifre değiştirme/sıfırlama, e-posta doğrulama, MFA ve OAuth/SSO.
type AuthHandler struct {
	auth      *auth.Service
	checker   rbac.Checker
	validate  *validator.Validate
	secure    bool // production'da cookie Secure flag'i
	turnstile turnstile.Verifier
	cache     appshared.Cache
}

// AuthDeps, AuthHandler'ın bağımlılıklarıdır.
type AuthDeps struct {
	Auth      *auth.Service
	Checker   rbac.Checker
	Validate  *validator.Validate
	Secure    bool
	Turnstile turnstile.Verifier
	Cache     appshared.Cache
}

// NewAuthHandler, handler'ı kurar.
func NewAuthHandler(d AuthDeps) *AuthHandler {
	return &AuthHandler{
		auth: d.Auth, checker: d.Checker, validate: d.Validate,
		secure: d.Secure, turnstile: d.Turnstile, cache: d.Cache,
	}
}

type loginRequest struct {
	Email          string `json:"email" validate:"omitempty,email" sanitize:"email"`
	Password       string `json:"password"`
	MFACode        string `json:"mfa_code"`
	MFAChallenge   string `json:"mfa_challenge"`
	TurnstileToken string `json:"turnstile_token"`
}

// Login, POST /auth/login — kimlik bilgilerini doğrular, token çifti döner ve
// refresh token'ı güvenli (HttpOnly, Secure, SameSite=Strict) cookie'ye yazar.
// MFA etkinse ilk istek 401 auth.mfa_required döner; istemci mfa_code ile tekrar dener.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if req.MFAChallenge == "" {
		if err := shared.VerifyTurnstile(c, h.turnstile, req.TurnstileToken); err != nil {
			return render.Error(c, err)
		}
		if req.Email == "" || req.Password == "" {
			return render.Error(c, auth.ErrInvalidCredentials)
		}
		if err := validation.Check(h.validate, &req); err != nil {
			return render.Error(c, err)
		}
	} else if req.MFACode == "" {
		return render.Error(c, auth.ErrInvalidMFACode)
	}

	tokens, err := h.auth.Login(c.Context(), auth.LoginCommand{
		Email: req.Email, Password: req.Password, MFACode: req.MFACode,
		MFAChallenge: req.MFAChallenge, ClientKey: c.IP(),
	})
	if err != nil {
		if challenge, ok := auth.MFAChallengeFrom(err); ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"type": "auth.mfa_required", "title": "İki adımlı doğrulama gerekli",
				"status": fiber.StatusUnauthorized, "detail": err.Error(),
				"mfa_challenge": challenge,
			})
		}
		return render.Error(c, err)
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	return render.JSON(c, fiber.StatusOK, tokens)
}

// Refresh, POST /auth/refresh — geçerli refresh token'dan yeni çift üretir
// (rotation). Tüketilmiş token yeniden kullanılırsa tüm oturumlar iptal edilir.
func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	refresh := h.refreshFromRequest(c)
	if refresh == "" {
		return render.Error(c, auth.ErrInvalidToken)
	}

	tokens, err := h.auth.Refresh(c.Context(), refresh)
	if err != nil {
		if errors.Is(err, auth.ErrTokenReuse) {
			h.clearRefreshCookie(c)
		}
		return render.Error(c, err)
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	return render.JSON(c, fiber.StatusOK, tokens)
}

// Logout, POST /auth/logout — access token'ı iptal eder, refresh'i tüketir ve
// cookie'yi temizler.
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	access := bearerToken(c)
	refresh := h.refreshFromRequest(c)
	_ = h.auth.Logout(c.Context(), access, refresh)
	h.clearRefreshCookie(c)
	return render.Message(c, fiber.StatusOK, "success.auth.logout", "çıkış yapıldı")
}

// Permissions, GET /auth/permissions — oturum açmış kullanıcının izin listesini döner.
func (h *AuthHandler) Permissions(c fiber.Ctx) error {
	role, _ := c.Locals(shared.LocalRole).(string)
	perms, err := h.checker.PermissionsFor(c.Context(), role)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, fiber.Map{
		"permissions": perms,
	})
}

// refreshFromRequest, refresh token'ı önce cookie'den, yoksa body'den alır.
func (h *AuthHandler) refreshFromRequest(c fiber.Ctx) string {
	refresh := c.Cookies("refresh_token")
	if refresh == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.Bind().Body(&body); err == nil {
			refresh = body.RefreshToken
		}
	}
	return refresh
}

// bearerToken, Authorization başlığından Bearer token'ı çıkarır.
func bearerToken(c fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

// setRefreshCookie, refresh token'ı güvenli cookie olarak yazar.
func (h *AuthHandler) setRefreshCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HTTPOnly: true,                           // XSS ile JS erişimini engeller
		Secure:   h.secure,                       // yalnızca HTTPS (production)
		SameSite: fiber.CookieSameSiteStrictMode, // CSRF'ye karşı
		Path:     "/",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}

// clearRefreshCookie, refresh cookie'sini geçersiz kılar.
func (h *AuthHandler) clearRefreshCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: fiber.CookieSameSiteStrictMode,
		Path:     "/",
	})
}
