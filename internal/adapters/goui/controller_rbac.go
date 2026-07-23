package goui

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/zatrano/gocore/internal/application/authz"
	dshared "github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

var (
	errForbiddenAction = errors.New("bu işlem için yetkiniz yok")
	errAuthzRequired   = errors.New("yetki servisi yapılandırılmamış")
	errSenderRequired  = errors.New("bildirim gönderici yapılandırılmamış")
	errUploadRequired  = errors.New("yükleme servisi yapılandırılmamış")
	errStorageRequired = errors.New("depolama yapılandırılmamış")
	errMissingUpload   = errors.New("en az bir dosya seçin")
	errFileTooLarge    = errors.New("izin verilen boyut aşıldı")
)

func requirePagePerm(ctx context.Context, p *Page, perm rbac.Permission) error {
	if p == nil || !p.Allowed(ctx, perm) {
		return errForbiddenAction
	}
	return nil
}

func userFacingError(err error) string {
	if err == nil {
		return ""
	}
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		return "Bir veya daha fazla alan geçersiz"
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return "Bir veya daha fazla alan geçersiz"
	}
	if de, ok := dshared.AsDomainError(err); ok {
		return de.Message
	}
	return err.Error()
}

func fieldErrorsFrom(err error) map[string]string {
	out := map[string]string{}
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		for _, f := range inv {
			out[f.Field] = f.Tag + " geçersiz"
		}
		return out
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		for _, fe := range verrs {
			name := fe.Field()
			if tag := fe.StructField(); tag != "" {
				// form tag tercih edilir; validator Field() exported adı döner
				name = strings.ToLower(fe.Field())
			}
			out[strings.ToLower(fe.Field())] = fe.Tag()
			_ = name
		}
	}
	return out
}
func toPermSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}

// ---------------------------------------------------------------------------
// roles
// ---------------------------------------------------------------------------

type rolesController struct {
	roles []authz.RoleInfo
}

func (c *rolesController) Mount(ctx context.Context, p *Page) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	roles, err := p.Deps.Authz.ListRoles(ctx)
	if err != nil {
		return err
	}
	c.roles = roles
	return nil
}

func (c *rolesController) Render(p *Page) (string, error) {
	type roleRow struct {
		Name        string
		Description string
		IsSystem    bool
		PermCount   string
		EditHref    string
	}
	rows := make([]roleRow, 0, len(c.roles))
	for _, role := range c.roles {
		rows = append(rows, roleRow{
			Name:        role.Name,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			PermCount:   strconv.Itoa(len(role.Permissions)),
			EditHref:    "/dashboard/rbac/roles/" + role.Name,
		})
	}
	return p.RenderView("pages.roles", map[string]any{
		"HasRoles": len(c.roles) > 0,
		"Roles":    rows,
	})
}

func (c *rolesController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	_ = payload
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if event == "roles.reload" {
		return c.Mount(ctx, p)
	}
	return nil
}

// ---------------------------------------------------------------------------
// role-new
// ---------------------------------------------------------------------------

type roleNewController struct {
	name        string
	description string
	permissions []authz.PermissionInfo
	selected    map[string]bool
	fieldErrs   map[string]string
}

func (c *roleNewController) Mount(ctx context.Context, p *Page) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	perms, err := p.Deps.Authz.ListPermissions(ctx)
	if err != nil {
		return err
	}
	c.permissions = perms
	if c.selected == nil {
		c.selected = map[string]bool{}
	}
	return nil
}

func (c *roleNewController) Render(p *Page) (string, error) {
	type permItem struct {
		Name     string
		Selected bool
	}
	perms := make([]permItem, 0, len(c.permissions))
	for _, perm := range c.permissions {
		perms = append(perms, permItem{Name: perm.Name, Selected: c.selected[perm.Name]})
	}
	return p.RenderView("pages.role_new", map[string]any{
		"Name":        c.name,
		"Description": c.description,
		"NameError":   viewFieldError(c.fieldErrs, "name"),
		"Permissions": perms,
	})
}

type createRoleForm struct {
	Name        string   `form:"name" validate:"required,min=2,max=32"`
	Description string   `form:"description" validate:"max=255"`
	Permissions []string `form:"permissions"`
}

func (c *roleNewController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	switch event {
	case "role.select_all":
		c.selected = map[string]bool{}
		for _, perm := range c.permissions {
			c.selected[perm.Name] = true
		}
		return nil
	case "role.clear_all":
		c.selected = map[string]bool{}
		return nil
	case "role.create":
		req := createRoleForm{
			Name:        payloadString(payload, "name"),
			Description: payloadString(payload, "description"),
			Permissions: payloadStrings(payload, "permissions"),
		}
		c.name = req.Name
		c.description = req.Description
		c.selected = toPermSet(req.Permissions)
		c.fieldErrs = nil
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				c.fieldErrs = fieldErrorsFrom(err)
				p.Error = userFacingError(err)
				return nil
			}
		}
		if p.Deps.Authz == nil {
			return errAuthzRequired
		}
		role, err := p.Deps.Authz.CreateRole(ctx, req.Name, req.Description, req.Permissions)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		p.Notice = "rol oluşturuldu"
		p.Redirect = "/dashboard/rbac/roles/" + role.Name
		return nil
	}
	return nil
}

// ---------------------------------------------------------------------------
// role-show
// ---------------------------------------------------------------------------

type roleShowController struct {
	role     authz.RoleInfo
	allPerms []authz.PermissionInfo
	selected map[string]bool
}

func (c *roleShowController) Mount(ctx context.Context, p *Page) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	name := ""
	if p.Params != nil {
		name = p.Params["name"]
	}
	role, err := p.Deps.Authz.GetRole(ctx, name)
	if err != nil {
		return err
	}
	perms, _ := p.Deps.Authz.ListPermissions(ctx)
	c.role = role
	c.allPerms = perms
	c.selected = toPermSet(role.Permissions)
	return nil
}

func (c *roleShowController) Render(p *Page) (string, error) {
	type permItem struct {
		Name     string
		Selected bool
	}
	perms := make([]permItem, 0, len(c.allPerms))
	for _, perm := range c.allPerms {
		perms = append(perms, permItem{Name: perm.Name, Selected: c.selected[perm.Name]})
	}
	return p.RenderView("pages.role_show", map[string]any{
		"RoleName":    c.role.Name,
		"Description": c.role.Description,
		"IsSystem":    c.role.IsSystem,
		"ShowDelete":  !c.role.IsSystem,
		"Permissions": perms,
	})
}

type updateRoleForm struct {
	Description string `form:"description" validate:"max=255"`
}

func (c *roleShowController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	name := c.role.Name
	if name == "" && p.Params != nil {
		name = p.Params["name"]
	}
	switch event {
	case "role.select_all":
		c.selected = map[string]bool{}
		for _, perm := range c.allPerms {
			c.selected[perm.Name] = true
		}
		return nil
	case "role.clear_all":
		c.selected = map[string]bool{}
		return nil
	case "role.update":
		req := updateRoleForm{Description: payloadString(payload, "description")}
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				p.Error = userFacingError(err)
				return nil
			}
		}
		role, err := p.Deps.Authz.UpdateRole(ctx, name, req.Description)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		c.role = role
		p.Notice = "rol güncellendi"
		return nil
	case "role.set_permissions":
		perms := payloadStrings(payload, "permissions")
		role, err := p.Deps.Authz.SetPermissions(ctx, name, perms)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		c.role = role
		c.selected = toPermSet(role.Permissions)
		p.Notice = "rol izinleri güncellendi"
		return nil
	case "role.delete":
		if err := p.Deps.Authz.DeleteRole(ctx, name); err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		p.Notice = "rol silindi"
		p.Redirect = "/dashboard/rbac/roles"
		return nil
	}
	return nil
}

// ---------------------------------------------------------------------------
// permissions
// ---------------------------------------------------------------------------

type permissionsController struct {
	permissions []authz.PermissionInfo
	formName    string
	formDesc    string
	fieldErrs   map[string]string
}

func (c *permissionsController) Mount(ctx context.Context, p *Page) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	perms, err := p.Deps.Authz.ListPermissions(ctx)
	if err != nil {
		return err
	}
	c.permissions = perms
	return nil
}

func (c *permissionsController) Render(p *Page) (string, error) {
	return p.RenderView("pages.permissions", map[string]any{
		"FormName":       c.formName,
		"FormDesc":       c.formDesc,
		"NameError":      viewFieldError(c.fieldErrs, "name"),
		"HasPermissions": len(c.permissions) > 0,
		"Permissions":    c.permissions,
	})
}

type createPermissionForm struct {
	Name        string `form:"name" validate:"required,min=3,max=64"`
	Description string `form:"description" validate:"max=255"`
}

type updatePermissionForm struct {
	Description string `form:"description" validate:"max=255"`
}

func (c *permissionsController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermRBACManage); err != nil {
		return err
	}
	if p.Deps.Authz == nil {
		return errAuthzRequired
	}
	switch event {
	case "permission.create":
		req := createPermissionForm{
			Name:        payloadString(payload, "name"),
			Description: payloadString(payload, "description"),
		}
		c.formName = req.Name
		c.formDesc = req.Description
		c.fieldErrs = nil
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				c.fieldErrs = fieldErrorsFrom(err)
				p.Error = userFacingError(err)
				return nil
			}
		}
		if _, err := p.Deps.Authz.CreatePermission(ctx, req.Name, req.Description); err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		c.formName, c.formDesc = "", ""
		p.Notice = "izin oluşturuldu"
		return c.Mount(ctx, p)
	case "permission.update":
		name := payloadString(payload, "name")
		req := updatePermissionForm{Description: payloadString(payload, "description")}
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				p.Error = userFacingError(err)
				return nil
			}
		}
		if _, err := p.Deps.Authz.UpdatePermission(ctx, name, req.Description); err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		p.Notice = "izin güncellendi"
		return c.Mount(ctx, p)
	}
	return nil
}
