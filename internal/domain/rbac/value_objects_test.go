package rbac_test

import (
	"testing"

	"github.com/zatrano/gocore/internal/domain/rbac"
)

var seededPermissionNames = []string{
	"users:list", "users:read", "users:activate", "users:delete", "users:restore",
	"users:role:change", "users:email:change:any",
	"uploads:create", "rbac:manage", "notifications:send", "notifications:settings",
	"payments:charge", "payments:list",
}

func TestParsePermissionName_SeededNames(t *testing.T) {
	for _, name := range seededPermissionNames {
		t.Run(name, func(t *testing.T) {
			if _, err := rbac.NewPermission(name, "test"); err != nil {
				t.Fatalf("NewPermission(%q): %v", name, err)
			}
		})
	}
}

func TestParsePermissionName_Invalid(t *testing.T) {
	for _, raw := range []string{"", "Admin", "no-colon", "bad:UPPER"} {
		if _, err := rbac.NewPermission(raw, "x"); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}
