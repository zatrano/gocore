package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
)

const headerIdempotencyKey = "Idempotency-Key"

// IdempotencyGuard, Idempotency-Key ile tüm API mutasyonlarını idempotent yapar.
// Authenticator'dan sonra çalışmalıdır (aktör kimliği için).
func IdempotencyGuard(svc *appidempotency.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !isMutatingMethod(c.Method()) || isIdempotencyExempt(c.Path()) {
			return c.Next()
		}
		key := strings.TrimSpace(c.Get(headerIdempotencyKey))
		if key == "" {
			return c.Next()
		}

		actorID := idempotencyActor(c)
		scope := apiScope(c)
		reqHash := appidempotency.HashRequest(c.Body())
		ctx := appidempotency.WithKey(c.Context(), scopedContextKey(actorID, key))
		c.SetContext(ctx)

		cached, stored, err := svc.RunHTTP(ctx, scope, key, actorID, reqHash, func() (*appidempotency.HTTPStoredResponse, error) {
			if err := c.Next(); err != nil {
				return nil, err
			}
			body := append([]byte(nil), c.Response().Body()...)
			if len(body) == 0 && c.Response().StatusCode() < 300 {
				body = []byte("null")
			}
			return &appidempotency.HTTPStoredResponse{
				StatusCode:  c.Response().StatusCode(),
				Body:        body,
				ContentType: string(c.Response().Header.ContentType()),
			}, nil
		})
		if err != nil {
			return render.Error(c, err)
		}
		if cached {
			return writeStoredResponse(c, stored)
		}
		return nil
	}
}

// Idempotency, context'e Idempotency-Key taşır (use-case düzeyi idempotency için).
func Idempotency() fiber.Handler {
	return func(c fiber.Ctx) error {
		key := strings.TrimSpace(c.Get(headerIdempotencyKey))
		if key == "" {
			return c.Next()
		}
		actorID := idempotencyActor(c)
		c.SetContext(appidempotency.WithKey(c.Context(), scopedContextKey(actorID, key)))
		return c.Next()
	}
}

func writeStoredResponse(c fiber.Ctx, stored *appidempotency.HTTPStoredResponse) error {
	if stored.ContentType != "" {
		c.Set(fiber.HeaderContentType, stored.ContentType)
	} else {
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	return c.Status(stored.StatusCode).Send(stored.Body)
}

func idempotencyActor(c fiber.Ctx) string {
	if id, _ := c.Locals(adapters.LocalUserID).(string); id != "" {
		return id
	}
	return c.IP()
}

func scopedContextKey(actorID, key string) string {
	if actorID == "" {
		return key
	}
	return actorID + ":" + key
}

func apiScope(c fiber.Ctx) string {
	return appidempotency.ScopeAPI + ":" + c.Method() + ":" + c.Path()
}

func isMutatingMethod(method string) bool {
	switch method {
	case fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete:
		return true
	default:
		return false
	}
}

func isIdempotencyExempt(path string) bool {
	switch path {
	case "/api/v1/payments/3ds/callback", "/api/v1/payments/webhook/iyzico":
		return true
	default:
		return strings.HasPrefix(path, "/api/v1/auth/oauth/")
	}
}
