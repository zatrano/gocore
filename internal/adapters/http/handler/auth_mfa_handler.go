package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
)

// MFASetup, POST /auth/mfa/setup — yeni TOTP sırrı üretir (henüz etkin değil).
// Yanıt gizli anahtar ve otpauth URI içerir (authenticator'a QR olarak girilir).
func (h *AuthHandler) MFASetup(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	res, err := h.auth.MFASetup(c.Context(), userID)
	if err != nil {
		return render.Error(c, err)
	}
	c.Set("Cache-Control", "no-store, private")
	c.Set("Pragma", "no-cache")
	return render.JSONWithMessage(c, fiber.StatusOK, "success.auth.mfa_setup",
		"iki adımlı doğrulama kurulumu başlatıldı", res)
}

type mfaCodeRequest struct {
	Code string `json:"code" validate:"required"`
}

// MFAEnable, POST /auth/mfa/enable — kurulumdaki kodu doğrular ve MFA'yı etkinleştirir.
func (h *AuthHandler) MFAEnable(c fiber.Ctx) error {
	var req mfaCodeRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	result, err := h.auth.MFAEnable(c.Context(), auth.EnableCommand{UserID: userID, Code: req.Code})
	if err != nil {
		return render.Error(c, err)
	}
	c.Set("Cache-Control", "no-store, private")
	c.Set("Pragma", "no-cache")
	return render.JSONWithMessage(c, fiber.StatusOK, "success.auth.mfa_enabled",
		"iki adımlı doğrulama etkinleştirildi", result)
}

// MFADisable, POST /auth/mfa/disable — geçerli kodla MFA'yı kapatır.
func (h *AuthHandler) MFADisable(c fiber.Ctx) error {
	var req mfaCodeRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	if err := h.auth.MFADisable(c.Context(), auth.DisableCommand{UserID: userID, Code: req.Code}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.mfa_disabled", "iki adımlı doğrulama kapatıldı")
}
