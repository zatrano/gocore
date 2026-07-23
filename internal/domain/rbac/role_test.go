package rbac_test

import (
	"errors"
	"testing"

	"github.com/zatrano/gocore/internal/domain/rbac"
)

func TestCreateRole_ValidName(t *testing.T) {
	r, err := rbac.CreateRole("editor", "İçerik editörü")
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if r.Name() != "editor" {
		t.Fatalf("name = %q", r.Name())
	}
	if r.IsSystem() {
		t.Fatal("custom role must not be system")
	}
}

func TestCreateRole_InvalidName(t *testing.T) {
	_, err := rbac.CreateRole("Admin", "bad")
	if err == nil {
		t.Fatal("expected error for uppercase name")
	}
}

func TestRole_EnsureDeletable(t *testing.T) {
	custom, _ := rbac.CreateRole("editor", "")
	system, _ := rbac.CreateSystemRole(rbac.RoleAdmin, "admin")

	if err := custom.EnsureDeletable(0); err != nil {
		t.Fatalf("custom role with no users should be deletable: %v", err)
	}
	if err := custom.EnsureDeletable(1); !errors.Is(err, rbac.ErrRoleInUse) {
		t.Fatalf("expected ErrRoleInUse, got %v", err)
	}
	if err := system.EnsureDeletable(0); !errors.Is(err, rbac.ErrSystemRoleImmutable) {
		t.Fatalf("expected ErrSystemRoleImmutable, got %v", err)
	}
}

func TestRole_ReplacePermissions(t *testing.T) {
	r, _ := rbac.CreateRole("editor", "")
	r.ReplacePermissions([]rbac.PermissionName{"users:list", "users:read"})
	if len(r.Permissions()) != 2 {
		t.Fatalf("permissions len = %d", len(r.Permissions()))
	}
}
