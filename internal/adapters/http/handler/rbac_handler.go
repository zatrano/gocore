package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	"github.com/zatrano/gocore/internal/application/authz"
)

// RBACHandler, rollerin ve izinlerin çalışma zamanı yönetimi için HTTP uç
// noktalarını sağlar (admin API'si). Tüm rotalar rbac:manage izni gerektirir.
type RBACHandler struct {
	svc      *authz.Service
	validate *validator.Validate
}

// NewRBACHandler, handler'ı kurar.
func NewRBACHandler(svc *authz.Service, validate *validator.Validate) *RBACHandler {
	return &RBACHandler{svc: svc, validate: validate}
}

// --- Response DTO'ları ---

type permissionView struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type roleView struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
}

func toRoleView(r authz.RoleInfo) roleView {
	perms := r.Permissions
	if perms == nil {
		perms = []string{}
	}
	return roleView{Name: r.Name, Description: r.Description, IsSystem: r.IsSystem, Permissions: perms}
}

// --- Request DTO'ları ---

type createRoleRequest struct {
	Name        string   `json:"name" validate:"required,min=2,max=32"`
	Description string   `json:"description" validate:"max=255"`
	Permissions []string `json:"permissions" validate:"dive,required"`
}

type updateRoleRequest struct {
	Description string `json:"description" validate:"max=255"`
}

type setPermissionsRequest struct {
	Permissions []string `json:"permissions" validate:"dive,required"`
}

type createPermissionRequest struct {
	Name        string `json:"name" validate:"required,min=3,max=64"`
	Description string `json:"description" validate:"max=255"`
}

type updatePermissionRequest struct {
	Description string `json:"description" validate:"max=255"`
}

// ListPermissions, GET /rbac/permissions — izin listesini döner.
func (h *RBACHandler) ListPermissions(c fiber.Ctx) error {
	perms, err := h.svc.ListPermissions(c.Context())
	if err != nil {
		return render.Error(c, err)
	}
	out := make([]permissionView, 0, len(perms))
	for _, p := range perms {
		out = append(out, permissionView{Name: p.Name, Description: p.Description})
	}
	return render.JSON(c, fiber.StatusOK, fiber.Map{"permissions": out})
}

// CreatePermission, POST /rbac/permissions — yeni izin oluşturur.
func (h *RBACHandler) CreatePermission(c fiber.Ctx) error {
	var req createPermissionRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	perm, err := h.svc.CreatePermission(c.Context(), req.Name, req.Description)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusCreated, "success.rbac.permission_created", "izin oluşturuldu", permissionView{
		Name: perm.Name, Description: perm.Description,
	})
}

// UpdatePermission, PATCH /rbac/permissions/:name — izin açıklamasını günceller.
func (h *RBACHandler) UpdatePermission(c fiber.Ctx) error {
	var req updatePermissionRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	perm, err := h.svc.UpdatePermission(c.Context(), c.Params("name"), req.Description)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.rbac.permission_updated", "izin güncellendi", permissionView{
		Name: perm.Name, Description: perm.Description,
	})
}

// ListRoles, GET /rbac/roles — tüm rolleri izinleriyle döner.
func (h *RBACHandler) ListRoles(c fiber.Ctx) error {
	roles, err := h.svc.ListRoles(c.Context())
	if err != nil {
		return render.Error(c, err)
	}
	out := make([]roleView, 0, len(roles))
	for _, r := range roles {
		out = append(out, toRoleView(r))
	}
	return render.JSON(c, fiber.StatusOK, fiber.Map{"roles": out})
}

// GetRole, GET /rbac/roles/:name — tek rolü izinleriyle döner.
func (h *RBACHandler) GetRole(c fiber.Ctx) error {
	role, err := h.svc.GetRole(c.Context(), c.Params("name"))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, toRoleView(role))
}

// CreateRole, POST /rbac/roles — yeni rol oluşturur.
func (h *RBACHandler) CreateRole(c fiber.Ctx) error {
	var req createRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	role, err := h.svc.CreateRole(c.Context(), req.Name, req.Description, req.Permissions)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusCreated, "success.rbac.role_created", "rol oluşturuldu", toRoleView(role))
}

// UpdateRole, PATCH /rbac/roles/:name — rol açıklamasını günceller.
func (h *RBACHandler) UpdateRole(c fiber.Ctx) error {
	var req updateRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	role, err := h.svc.UpdateRole(c.Context(), c.Params("name"), req.Description)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.rbac.role_updated", "rol güncellendi", toRoleView(role))
}

// SetPermissions, PUT /rbac/roles/:name/permissions — rolün izin kümesini değiştirir.
func (h *RBACHandler) SetPermissions(c fiber.Ctx) error {
	var req setPermissionsRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	role, err := h.svc.SetPermissions(c.Context(), c.Params("name"), req.Permissions)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.rbac.permissions_updated", "rol izinleri güncellendi", toRoleView(role))
}

// DeleteRole, DELETE /rbac/roles/:name — sistem-olmayan bir rolü siler.
func (h *RBACHandler) DeleteRole(c fiber.Ctx) error {
	if err := h.svc.DeleteRole(c.Context(), c.Params("name")); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.rbac.role_deleted", "rol silindi")
}
