package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// SettingsHandler, uygulama ayarları HTTP uç noktalarını sağlar.
type SettingsHandler struct {
	settings *appsettings.Service
	notify   config.Notify
	payment  config.Payment
	validate *validator.Validate
}

// NewSettingsHandler, handler'ı kurar.
func NewSettingsHandler(svc *appsettings.Service, notify config.Notify, payment config.Payment, validate *validator.Validate) *SettingsHandler {
	return &SettingsHandler{settings: svc, notify: notify, payment: payment, validate: validate}
}

func (h *SettingsHandler) smsStatus() appsettings.SMSIntegrationStatus {
	return adaptershared.SMSIntegrationStatus(h.notify)
}

func (h *SettingsHandler) paymentStatus() appsettings.PaymentIntegrationStatus {
	return adaptershared.PaymentIntegrationStatus(h.payment)
}

// ListSMS, GET /settings/sms — SMS sağlayıcı listesi.
func (h *SettingsHandler) ListSMS(c fiber.Ctx) error {
	st := h.smsStatus()
	rows, active, err := h.settings.ListSMSProviders(c.Context(), st)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, adaptershared.NewSMSListPayload(rows, active, st))
}

// GetSMSProvider, GET /settings/sms/:provider — tek SMS sağlayıcı detayı.
func (h *SettingsHandler) GetSMSProvider(c fiber.Ctx) error {
	name := c.Params("provider")
	row, err := h.settings.GetSMSProvider(c.Context(), name, h.smsStatus())
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, adaptershared.NewSMSProviderPayload(row, h.smsStatus()))
}

type selectSMSProviderRequest struct {
	Provider  string `json:"provider" validate:"required,oneof=netgsm iletimerkezi"`
	SetActive bool   `json:"set_active"`
}

// CreateSMS, POST /settings/sms — sağlayıcı seçimi (isteğe bağlı aktifleştirme).
func (h *SettingsHandler) CreateSMS(c fiber.Ctx) error {
	var req selectSMSProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if req.SetActive {
		if err := adaptershared.ValidateSMSActivation(req.Provider, h.smsStatus()); err != nil {
			return render.Error(c, err)
		}
		if err := h.settings.SetSMSActiveProvider(c.Context(), req.Provider); err != nil {
			return render.Error(c, err)
		}
	}
	row, err := h.settings.GetSMSProvider(c.Context(), req.Provider, h.smsStatus())
	if err != nil {
		return render.Error(c, err)
	}
	payload := adaptershared.NewSMSProviderPayload(row, h.smsStatus())
	if req.SetActive {
		return render.JSONWithMessage(c, fiber.StatusCreated,
			"success.settings.sms_provider_updated", "SMS sağlayıcısı güncellendi", payload)
	}
	return render.JSON(c, fiber.StatusCreated, payload)
}

// UpdateSMSProviderByName, PATCH /settings/sms/:provider — aktif SMS sağlayıcısını günceller.
func (h *SettingsHandler) UpdateSMSProviderByName(c fiber.Ctx) error {
	name := c.Params("provider")
	if err := adaptershared.ValidateSMSActivation(name, h.smsStatus()); err != nil {
		return render.Error(c, err)
	}
	if err := h.settings.SetSMSActiveProvider(c.Context(), name); err != nil {
		return render.Error(c, err)
	}
	row, err := h.settings.GetSMSProvider(c.Context(), name, h.smsStatus())
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK,
		"success.settings.sms_provider_updated", "SMS sağlayıcısı güncellendi",
		adaptershared.NewSMSProviderPayload(row, h.smsStatus()))
}

// ListPayment, GET /settings/payment — ödeme sağlayıcı listesi.
func (h *SettingsHandler) ListPayment(c fiber.Ctx) error {
	view, err := h.settings.GetPaymentView(c.Context(), h.paymentStatus())
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}

// GetPaymentProvider, GET /settings/payment/:provider — tek ödeme sağlayıcı detayı.
func (h *SettingsHandler) GetPaymentProvider(c fiber.Ctx) error {
	name := c.Params("provider")
	row, err := h.settings.GetPaymentProvider(c.Context(), name, h.paymentStatus())
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, adaptershared.NewPaymentProviderPayload(row))
}

type selectPaymentProviderRequest struct {
	Provider  string `json:"provider" validate:"required,oneof=iyzico moka"`
	SetActive bool   `json:"set_active"`
}

// CreatePayment, POST /settings/payment — sağlayıcı seçimi (isteğe bağlı aktifleştirme).
func (h *SettingsHandler) CreatePayment(c fiber.Ctx) error {
	var req selectPaymentProviderRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if req.SetActive {
		if err := adaptershared.ValidatePaymentActivation(req.Provider, h.paymentStatus()); err != nil {
			return render.Error(c, err)
		}
		if err := h.settings.SetPaymentActiveProvider(c.Context(), req.Provider); err != nil {
			return render.Error(c, err)
		}
	}
	row, err := h.settings.GetPaymentProvider(c.Context(), req.Provider, h.paymentStatus())
	if err != nil {
		return render.Error(c, err)
	}
	payload := adaptershared.NewPaymentProviderPayload(row)
	if req.SetActive {
		return render.JSONWithMessage(c, fiber.StatusCreated,
			"success.settings.payment_provider_updated", "Ödeme sağlayıcısı güncellendi", payload)
	}
	return render.JSON(c, fiber.StatusCreated, payload)
}

// UpdatePaymentProvider, PATCH /settings/payment/:provider — aktif ödeme sağlayıcısını günceller.
func (h *SettingsHandler) UpdatePaymentProvider(c fiber.Ctx) error {
	name := c.Params("provider")
	if err := adaptershared.ValidatePaymentActivation(name, h.paymentStatus()); err != nil {
		return render.Error(c, err)
	}
	if err := h.settings.SetPaymentActiveProvider(c.Context(), name); err != nil {
		return render.Error(c, err)
	}
	row, err := h.settings.GetPaymentProvider(c.Context(), name, h.paymentStatus())
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK,
		"success.settings.payment_provider_updated", "Ödeme sağlayıcısı güncellendi",
		adaptershared.NewPaymentProviderPayload(row))
}
