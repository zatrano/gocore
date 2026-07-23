package rbac_test

import (
	"context"
	"testing"

	"github.com/zatrano/gocore/pkg/rbac"
)

func allTestPermissions() []rbac.Permission {
	out := make([]rbac.Permission, 0, len(rbac.Catalog()))
	for _, def := range rbac.Catalog() {
		out = append(out, def.Permission)
	}
	return out
}

func newChecker() *rbac.StaticChecker {
	return rbac.NewStaticChecker(map[string][]rbac.Permission{
		"admin": allTestPermissions(),
		"user":  {},
	})
}

func TestStaticChecker_AdminHasAll(t *testing.T) {
	c := newChecker()
	for _, p := range allTestPermissions() {
		ok, err := c.Allows(context.Background(), "admin", p)
		if err != nil || !ok {
			t.Fatalf("admin should have %s (ok=%v err=%v)", p, ok, err)
		}
	}
}

func TestStaticChecker_UserHasNone(t *testing.T) {
	c := newChecker()
	for _, p := range allTestPermissions() {
		ok, _ := c.Allows(context.Background(), "user", p)
		if ok {
			t.Fatalf("user should not have %s", p)
		}
	}
}

func TestStaticChecker_UnknownRole(t *testing.T) {
	c := newChecker()
	ok, _ := c.Allows(context.Background(), "ghost", rbac.PermUsersList)
	if ok {
		t.Fatal("unknown role should have no permissions")
	}
}

func TestIsSelf(t *testing.T) {
	if !rbac.IsSelf("u1", "u1") {
		t.Fatal("same id should be self")
	}
	if rbac.IsSelf("u1", "u2") {
		t.Fatal("different id should not be self")
	}
	if rbac.IsSelf("", "") {
		t.Fatal("empty actor should not be self")
	}
}
