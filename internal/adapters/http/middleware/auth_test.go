package middleware_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/middleware"
	"github.com/zatrano/gocore/internal/adapters/http/problem"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/pkg/rbac"
)

type stubVerifier struct {
	claims auth.Claims
	err    error
}

func (v stubVerifier) Verify(context.Context, string) (auth.Claims, error) {
	return v.claims, v.err
}

type stubChecker map[string]map[rbac.Permission]bool

func (c stubChecker) Allows(_ context.Context, role string, p rbac.Permission) (bool, error) {
	if perms, ok := c[role]; ok {
		return perms[p], nil
	}
	return false, nil
}

func (c stubChecker) PermissionsFor(_ context.Context, role string) ([]string, error) {
	perms := c[role]
	out := make([]string, 0, len(perms))
	for p, ok := range perms {
		if ok {
			out = append(out, string(p))
		}
	}
	return out, nil
}

func TestAuthenticator_missingAndInvalid(t *testing.T) {
	app := fiber.New()
	app.Get("/x", middleware.Authenticator(stubVerifier{}), func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/x", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Basic abc")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("scheme status=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAuthenticator_okSetsLocals(t *testing.T) {
	app := fiber.New()
	var gotID, gotRole string
	app.Get("/x", middleware.Authenticator(stubVerifier{
		claims: auth.Claims{UserID: "u1", Role: "admin", Email: "a@b.co"},
	}), func(c fiber.Ctx) error {
		gotID, _ = c.Locals(adapters.LocalUserID).(string)
		gotRole, _ = c.Locals(adapters.LocalRole).(string)
		return c.SendString("ok")
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || gotID != "u1" || gotRole != "admin" {
		t.Fatalf("status=%d id=%s role=%s", resp.StatusCode, gotID, gotRole)
	}
}

func TestRequirePermission_allowsAndDenies(t *testing.T) {
	checker := stubChecker{
		"admin": {rbac.PermContactsList: true},
		"user":  {},
	}
	app := fiber.New()
	app.Get("/contacts",
		func(c fiber.Ctx) error {
			c.Locals(adapters.LocalRole, "user")
			return c.Next()
		},
		middleware.RequirePermission(checker, rbac.PermContactsList),
		func(c fiber.Ctx) error { return c.SendString("ok") },
	)
	app.Get("/contacts-admin",
		func(c fiber.Ctx) error {
			c.Locals(adapters.LocalRole, "admin")
			return c.Next()
		},
		middleware.RequirePermission(checker, rbac.PermContactsList),
		func(c fiber.Ctx) error { return c.SendString("ok") },
	)

	deny, err := app.Test(httptest.NewRequest(http.MethodGet, "/contacts", nil))
	if err != nil {
		t.Fatal(err)
	}
	if deny.StatusCode != 403 {
		t.Fatalf("deny status=%d", deny.StatusCode)
	}
	body, _ := io.ReadAll(deny.Body)
	_ = deny.Body.Close()
	var p problem.Problem
	_ = json.Unmarshal(body, &p)
	if p.Code != "auth.forbidden" {
		t.Fatalf("problem=%+v body=%s", p, body)
	}

	allow, err := app.Test(httptest.NewRequest(http.MethodGet, "/contacts-admin", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = allow.Body.Close()
	if allow.StatusCode != 200 {
		t.Fatalf("allow status=%d", allow.StatusCode)
	}
}

func TestRequirePermission_anyOf(t *testing.T) {
	checker := stubChecker{
		"ops": {rbac.PermPaymentsList: true},
	}
	app := fiber.New()
	app.Get("/pay",
		func(c fiber.Ctx) error {
			c.Locals(adapters.LocalRole, "ops")
			return c.Next()
		},
		middleware.RequirePermission(checker, rbac.PermPaymentsCharge, rbac.PermPaymentsList),
		func(c fiber.Ctx) error { return c.SendString("ok") },
	)
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/pay", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
