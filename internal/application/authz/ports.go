package authz

import (
	"context"

	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
)

// RoleInfo, bir rolün uygulama katmanı DTO'sudur (API/GoUI yanıtları).
type RoleInfo struct {
	Name        string
	Description string
	IsSystem    bool
	Permissions []string
}

// PermissionInfo, izin kataloğunun uygulama katmanı DTO'sudur.
type PermissionInfo struct {
	Name        string
	Description string
}

func toRoleInfo(r *domainrbac.Role) RoleInfo {
	return RoleInfo{
		Name:        r.Name().String(),
		Description: r.Description(),
		IsSystem:    r.IsSystem(),
		Permissions: r.PermissionNames(),
	}
}

func toPermissionInfo(p domainrbac.Permission) PermissionInfo {
	return PermissionInfo{
		Name:        p.Name().String(),
		Description: p.Description(),
	}
}

// RoleExistsChecker, user bounded context'inin RoleChecker portunu gerçekler.
type RoleExistsChecker struct {
	repo domainrbac.Repository
}

// NewRoleExistsChecker, RoleExistsChecker kurar.
func NewRoleExistsChecker(repo domainrbac.Repository) *RoleExistsChecker {
	return &RoleExistsChecker{repo: repo}
}

// RoleExists, rol adının tanımlı olup olmadığını döner.
func (c *RoleExistsChecker) RoleExists(ctx context.Context, role string) (bool, error) {
	name, err := domainrbac.ParseRoleName(role)
	if err != nil {
		return false, nil //nolint:nilerr // geçersiz rol adı: yok say
	}
	return c.repo.Exists(ctx, name)
}
