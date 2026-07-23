// Package handler, HTTP isteklerini uygulama use-case'lerine bağlayan Fiber
// handler'larını içerir.
package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
)

// Pinger, altyapı bağımlılıklarının sağlığını kontrol eden minimal arayüz
// (ör. *pgxpool.Pool).
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler, liveness/readiness/health probe'larını sağlar. Kubernetes ve
// yük dengeleyiciler bu endpoint'leri kullanır.
type HealthHandler struct {
	db      Pinger
	version string
	started time.Time
}

// NewHealthHandler, handler'ı kurar.
func NewHealthHandler(db Pinger, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version, started: time.Now()}
}

// Live, liveness probe: süreç ayakta mı? Bağımlılık kontrol etmez (sadece
// process'in yanıt verebildiğini gösterir).
func (h *HealthHandler) Live(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// Ready, readiness probe: trafiği kabul etmeye hazır mı? Kritik bağımlılıkları
// (DB) kısa timeout ile kontrol eder.
func (h *HealthHandler) Ready(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unavailable",
			"checks": fiber.Map{"database": "down"},
		})
	}
	return c.JSON(fiber.Map{
		"status": "ready",
		"checks": fiber.Map{"database": "up"},
	})
}

// Health, genel sağlık + meta bilgisi döner.
func (h *HealthHandler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": h.version,
		"uptime":  time.Since(h.started).String(),
	})
}
