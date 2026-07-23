package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	"github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
)

type changePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ChangePassword, POST /auth/change-password — oturum açmış kullanıcının şifresini
// mevcut şifresiyle değiştirir; tüm oturumlar iptal edilir.
func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	var req changePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	userID, _ := c.Locals(shared.LocalUserID).(string)
	err := h.auth.ChangePassword(c.Context(), auth.ChangePasswordCommand{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.password_changed", "şifreniz değiştirildi")
}

type forgotPasswordRequest struct {
	Email          string `json:"email" validate:"required,email" sanitize:"email"`
	TurnstileToken string `json:"turnstile_token"`
}

// ForgotPassword, POST /auth/forgot-password — sıfırlama bağlantısı gönderir.
// Numaralandırmaya karşı her zaman jenerik başarı döner.
func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := shared.VerifyTurnstile(c, h.turnstile, req.TurnstileToken); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if err := h.auth.ForgotPassword(c.Context(), auth.ForgotPasswordCommand{Email: req.Email}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.reset_sent",
		"eğer bu e-posta kayıtlıysa, sıfırlama bağlantısı gönderildi")
}

type resetPasswordRequest struct {
	Token          string `json:"token" validate:"required"`
	NewPassword    string `json:"new_password" validate:"required,min=8"`
	TurnstileToken string `json:"turnstile_token"`
}

// ResetPassword, POST /auth/reset-password — geçerli token ile yeni şifre belirler.
func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := shared.VerifyTurnstile(c, h.turnstile, req.TurnstileToken); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	err := h.auth.ResetPassword(c.Context(), auth.ResetPasswordCommand{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.password_reset", "şifreniz sıfırlandı")
}

type verifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmail, POST /auth/verify-email — e-posta doğrulama token'ını işler.
func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req verifyEmailRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if err := h.auth.VerifyEmail(c.Context(), auth.VerifyCommand{Token: req.Token}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.email_verified", "e-posta adresiniz doğrulandı")
}

type resendVerificationRequest struct {
	Email          string `json:"email" validate:"required,email" sanitize:"email"`
	TurnstileToken string `json:"turnstile_token"`
}

// ResendVerification, POST /auth/resend-verification — doğrulama e-postasını
// yeniden gönderir (jenerik başarı).
func (h *AuthHandler) ResendVerification(c fiber.Ctx) error {
	var req resendVerificationRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := shared.VerifyTurnstile(c, h.turnstile, req.TurnstileToken); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if err := h.auth.ResendVerification(c.Context(), auth.ResendCommand{Email: req.Email}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.auth.verification_sent",
		"eğer bu e-posta kayıtlıysa, doğrulama bağlantısı gönderildi")
}
