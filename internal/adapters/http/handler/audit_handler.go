package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appaudit "github.com/zatrano/gocore/internal/application/audit"
)

// AuditHandler, denetim kaydı API uç noktalarını sunar.
type AuditHandler struct {
	audit *appaudit.Service
}

// NewAuditHandler, handler'ı kurar.
func NewAuditHandler(audit *appaudit.Service) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// ListLogs, GET /api/v1/audit/logs — denetim kaydı listesi.
func (h *AuditHandler) ListLogs(c fiber.Ctx) error {
	page, err := h.audit.List(c.Context(), appaudit.ListQuery{
		Action: c.Query("action"), Resource: c.Query("resource"), Actor: c.Query("actor"),
		Page: adapters.ParsePage(c.Query("page")), Limit: adapters.ParseLimit(c.Query("limit")),
		Ascending: c.Query("order") == "asc",
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, page)
}

// GetLog, GET /api/v1/audit/logs/:id — denetim kaydı detayı.
func (h *AuditHandler) GetLog(c fiber.Ctx) error {
	view, err := h.audit.Get(c.Context(), c.Params("id"))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}
