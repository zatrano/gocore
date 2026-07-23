// Package middleware, HTTP katmanı için çapraz-kesen (cross-cutting) davranışları
// sağlar: correlation ID, yapısal loglama ve kimlik doğrulama.
package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/zatrano/gocore/internal/infrastructure/logger"
)

// HeaderCorrelationID, correlation ID başlığının adı.
const HeaderCorrelationID = "X-Correlation-ID"

// HeaderRequestID, istek kimliği başlığının adı (Fiber requestid ile uyumlu).
const HeaderRequestID = "X-Request-ID"

// Correlation, her isteğe bir correlation ID atar. X-Correlation-ID yoksa
// X-Request-ID kullanılır; o da yoksa yeni UUID üretilir. Yanıt başlığına
// yazılır ve Go context'ine aktarılır.
func Correlation() fiber.Handler {
	return func(c fiber.Ctx) error {
		requestID := c.Get(HeaderRequestID)
		cid := c.Get(HeaderCorrelationID)
		if cid == "" {
			cid = requestID
		}
		if cid == "" {
			cid = uuid.NewString()
		}
		if requestID == "" {
			requestID = cid
		}

		c.Set(HeaderCorrelationID, cid)
		c.Set(HeaderRequestID, requestID)

		ctx := logger.WithCorrelationID(c.Context(), cid)
		ctx = logger.WithRequestID(ctx, requestID)
		ctx = logger.WithRequestClient(ctx, c.IP(), c.Get(fiber.HeaderUserAgent))
		c.SetContext(ctx)
		return c.Next()
	}
}

// RequestLogger, her isteği yapısal olarak loglar (metod, yol, durum, süre).
func RequestLogger(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()

		attrs := []any{
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("ip", c.IP()),
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}

		// context'ten correlation ID otomatik eklenir (contextHandler).
		switch {
		case status >= 500:
			log.ErrorContext(c.Context(), "http request", attrs...)
		case status >= 400:
			log.WarnContext(c.Context(), "http request", attrs...)
		default:
			log.InfoContext(c.Context(), "http request", attrs...)
		}
		return err
	}
}
