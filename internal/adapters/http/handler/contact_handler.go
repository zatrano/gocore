package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/validation"
)

// ContactHandler, public iletişim formu ve admin inbox JSON API handler'ıdır.
type ContactHandler struct {
	contacts  *appcontact.Service
	validate  *validator.Validate
	turnstile turnstile.Verifier
}

// ContactDeps, ContactHandler bağımlılıklarını gruplar.
type ContactDeps struct {
	Contacts  *appcontact.Service
	Validate  *validator.Validate
	Turnstile turnstile.Verifier
}

// NewContactHandler, handler'ı kurar.
func NewContactHandler(d ContactDeps) *ContactHandler {
	return &ContactHandler{
		contacts: d.Contacts, validate: d.Validate, turnstile: d.Turnstile,
	}
}

type contactAPIRequest struct {
	Name    string `json:"name" validate:"required,min=2,max=100"`
	Email   string `json:"email" validate:"required,email" sanitize:"email"`
	Message string `json:"message" validate:"required,min=5,max=2000"`
	Locale  string `json:"locale" validate:"omitempty,max=10"`
}

// Submit, POST /api/v1/contact — iletişim mesajını kaydeder ve e-posta kuyruğuna alır.
func (h *ContactHandler) Submit(c fiber.Ctx) error {
	var req contactAPIRequest
	if err := c.Bind().JSON(&req); err != nil {
		return render.Error(c, err)
	}
	if err := adapters.VerifyTurnstile(c, h.turnstile, c.Get("X-Turnstile-Token")); err != nil {
		if token := c.FormValue("cf-turnstile-response"); token != "" {
			if err2 := adapters.VerifyTurnstile(c, h.turnstile, token); err2 != nil {
				return render.Error(c, err2)
			}
		} else {
			return render.Error(c, err)
		}
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	locale := req.Locale
	if locale == "" {
		locale = string(i18n.LocaleFromContext(c.Context()))
	}
	view, err := h.contacts.Submit(c.Context(), appcontact.SubmitCommand{
		Name: req.Name, Email: req.Email, Message: req.Message, Locale: locale,
		IP: c.IP(), UserAgent: string(c.Request().Header.UserAgent()),
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusCreated, fiber.Map{
		"id": view.ID,
		"message": i18n.T(c.Context(), "public.contact.success",
			"Mesajınız alındı, en kısa sürede dönüş yapacağız."),
	})
}

// List, GET /api/v1/contacts — iletişim mesajı listesi.
func (h *ContactHandler) List(c fiber.Ctx) error {
	page, err := h.contacts.List(c.Context(), appcontact.ListQuery{
		UnreadOnly: c.Query("unread") == "1" || c.Query("unread") == "true",
		Page:       adapters.ParsePage(c.Query("page")),
		Limit:      adapters.ParseLimit(c.Query("limit")),
		Ascending:  c.Query("order") == "asc",
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, page)
}

// Get, GET /api/v1/contacts/:id — iletişim mesajı detayı.
func (h *ContactHandler) Get(c fiber.Ctx) error {
	view, err := h.contacts.Get(c.Context(), c.Params("id"))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}

// MarkRead, POST /api/v1/contacts/:id/read — mesajı okundu işaretler.
func (h *ContactHandler) MarkRead(c fiber.Ctx) error {
	view, err := h.contacts.MarkRead(c.Context(), appcontact.MarkReadCommand{ID: c.Params("id")})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}
