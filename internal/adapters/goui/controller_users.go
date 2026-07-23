package goui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	appauthz "github.com/zatrano/gocore/internal/application/authz"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
)

// ---------------------------------------------------------------------------
// Users list
// ---------------------------------------------------------------------------

type usersListController struct {
	page pagination.Page[appuser.View]
}

func (c *usersListController) listQuery(p *Page) appuser.ListQuery {
	pageNum, limit := parsePageLimit(p)
	q := appuser.ListQuery{
		ActorRole: actorRole(p),
		Role:      pageQuery(p, "role", ""),
		Search:    pageQuery(p, "search", ""),
		Deleted:   pageQuery(p, "deleted", ""),
		Page:      pageNum,
		Limit:     limit,
		Ascending: pageQuery(p, "order", "") == "asc",
	}
	if activeStr := pageQuery(p, "active", ""); activeStr != "" {
		active := activeStr == "true"
		q.Active = &active
	}
	return q
}

func (c *usersListController) reload(ctx context.Context, p *Page) error {
	if err := requireAccess(p.Deps.Users.Access().CanListUsers(ctx, actorRole(p))); err != nil {
		return err
	}
	if p.Deps.Users == nil {
		return errors.New("kullanıcı listesi servisi yapılandırılmamış")
	}
	page, err := p.Deps.Users.List(ctx, c.listQuery(p))
	if err != nil {
		return err
	}
	c.page = page
	return nil
}

func (c *usersListController) Mount(ctx context.Context, p *Page) error {
	if err := c.reload(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
	}
	return nil
}

// usersListItem, kullanıcı listesi satırı.
type usersListItem struct {
	ID         string
	Name       string
	Email      string
	Role       string
	Status     string
	DetailHref string
}

func (c *usersListController) Render(p *Page) (string, error) {
	q := c.listQuery(p)
	items := make([]usersListItem, 0, len(c.page.Items))
	for _, u := range c.page.Items {
		status := "Pasif"
		if u.Deleted {
			status = "Silindi"
		} else if u.Active {
			status = "Aktif"
		}
		items = append(items, usersListItem{
			ID: u.ID, Name: u.Name, Email: u.Email,
			Role: u.Role, Status: status,
			DetailHref: "/dashboard/users/" + u.ID,
		})
	}
	deletedOptions := viewSelectOptions([][2]string{
		{"", "Canlı kayıtlar"}, {"only", "Yalnızca silinenler"}, {"all", "Tümü"},
	}, q.Deleted)
	return p.RenderView("pages.users", map[string]any{
		"Search":         q.Search,
		"Role":           q.Role,
		"DeletedOptions": deletedOptions,
		"LimitOptions":   viewLimitOptions(q.Limit),
		"ExportLinks": viewExportLinks("/dashboard/users/export", map[string]string{
			"role": q.Role, "search": q.Search, "deleted": q.Deleted,
			"order": pageQuery(p, "order", ""), "active": pageQuery(p, "active", ""),
		}),
		"Empty": len(c.page.Items) == 0,
		"Items": items,
		"Pages": viewPagination(c.page.Page, c.page.TotalPages),
	})
}

func (c *usersListController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "users.search":
		v := ""
		if raw, ok := payload["value"]; ok {
			v = strings.TrimSpace(fmt.Sprint(raw))
		} else {
			v = payloadString(payload, "search")
		}
		setQuery(p, "search", v)
		setQuery(p, "page", "1")
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	case "users.role":
		v := ""
		if raw, ok := payload["value"]; ok {
			v = strings.TrimSpace(fmt.Sprint(raw))
		} else {
			v = payloadString(payload, "role")
		}
		setQuery(p, "role", v)
		setQuery(p, "page", "1")
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	case "users.deleted":
		v := ""
		if raw, ok := payload["value"]; ok {
			v = strings.TrimSpace(fmt.Sprint(raw))
		} else {
			v = payloadString(payload, "deleted")
		}
		setQuery(p, "deleted", v)
		setQuery(p, "page", "1")
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	case "users.limit":
		v := ""
		if raw, ok := payload["value"]; ok {
			v = strings.TrimSpace(fmt.Sprint(raw))
		} else {
			v = payloadString(payload, "limit")
		}
		setQuery(p, "limit", v)
		setQuery(p, "page", "1")
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	case "users.page":
		n := payloadPage(payload, 1)
		setQuery(p, "page", strconv.Itoa(n))
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	case "users.clear":
		p.Query = map[string]string{}
		if err := c.reload(ctx, p); err != nil {
			p.Error = accountDisplayErr(err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// User new
// ---------------------------------------------------------------------------

type adminRegisterForm struct {
	Email    string `form:"email" validate:"required,email" sanitize:"email"`
	Name     string `form:"name" validate:"required,min=2,max=100"`
	Password string `form:"password" validate:"required,min=8,max=128"`
	Phone    string `form:"phone" validate:"omitempty,phone" sanitize:"phone"`
	Role     string `form:"role" validate:"required"`
}

type userNewController struct {
	roles       []appauthz.RoleInfo
	form        adminRegisterForm
	fieldErrors map[string]string
}

func (c *userNewController) loadRoles(ctx context.Context, p *Page) error {
	if p.Deps.Authz == nil {
		return errors.New("yetkilendirme servisi yapılandırılmamış")
	}
	roles, err := p.Deps.Authz.ListRoles(ctx)
	if err != nil {
		return err
	}
	c.roles = roles
	return nil
}

func (c *userNewController) Mount(ctx context.Context, p *Page) error {
	if err := requireAccess(p.Deps.Users.Access().CanListUsers(ctx, actorRole(p))); err != nil {
		p.Error = accountDisplayErr(err)
		return nil
	}
	if c.form.Role == "" {
		c.form.Role = "user"
	}
	if err := c.loadRoles(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
	}
	return nil
}

func (c *userNewController) Render(p *Page) (string, error) {
	roleOpts := make([]ViewOption, 0, len(c.roles))
	for _, r := range c.roles {
		roleOpts = append(roleOpts, ViewOption{Value: r.Name, Label: r.Name, Selected: r.Name == c.form.Role})
	}
	return p.RenderView("pages.user_new", map[string]any{
		"FormName":    c.form.Name,
		"FormEmail":   c.form.Email,
		"FormPhone":   c.form.Phone,
		"RoleOptions": roleOpts,
		"ErrName":     viewFieldError(c.fieldErrors, "name"),
		"ErrEmail":    viewFieldError(c.fieldErrors, "email"),
		"ErrPhone":    viewFieldError(c.fieldErrors, "phone"),
		"ErrPassword": viewFieldError(c.fieldErrors, "password"),
		"ErrRole":     viewFieldError(c.fieldErrors, "role"),
	})
}

func (c *userNewController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if event != "users.create" {
		return nil
	}
	if err := requireAccess(p.Deps.Users.Access().CanListUsers(ctx, actorRole(p))); err != nil {
		p.Error = accountDisplayErr(err)
		return nil
	}
	c.form = adminRegisterForm{
		Email:    payloadString(payload, "email"),
		Name:     payloadString(payload, "name"),
		Password: payloadString(payload, "password"),
		Phone:    payloadString(payload, "phone"),
		Role:     payloadString(payload, "role"),
	}
	if c.form.Role == "" {
		c.form.Role = "user"
	}
	_ = c.loadRoles(ctx, p)
	if err := validateDeps(p, &c.form); err != nil {
		c.fieldErrors = accountFieldErrors(ctx, err)
		p.Error = accountDisplayErr(err)
		return nil
	}
	if p.Deps.Users == nil {
		return errors.New("kayıt servisi yapılandırılmamış")
	}
	view, err := p.Deps.Users.Register(ctx, appuser.RegisterCommand{
		Email: c.form.Email, Name: c.form.Name, Password: c.form.Password,
		Role: c.form.Role, Phone: c.form.Phone, PreferredLocale: preferredLocale(p),
		AllowPrivilegedRole: true,
	})
	if err != nil {
		c.fieldErrors = accountFieldErrors(ctx, err)
		p.Error = accountDisplayErr(err)
		return nil
	}
	p.Notice = "kullanıcı başarıyla kaydedildi"
	p.Redirect = "/dashboard/users/" + view.ID
	return nil
}

// ---------------------------------------------------------------------------
// User show
// ---------------------------------------------------------------------------

type userChangeRoleForm struct {
	Role string `form:"role" validate:"required"`
}
type userChangeEmailForm struct {
	Email string `form:"email" validate:"required,email" sanitize:"email"`
}
type userChangeNameForm struct {
	Name string `form:"name" validate:"required,min=2,max=100"`
}
type userChangePhoneForm struct {
	Phone string `form:"phone" validate:"omitempty,phone" sanitize:"phone"`
}

type userShowPerms struct {
	CanReadUser         bool
	CanChangeRole       bool
	CanActivate         bool
	CanChangeProfileAny bool
	CanDelete           bool
	CanRestore          bool
}

type userShowController struct {
	profile      appuser.View
	perms        userShowPerms
	roles        []appauthz.RoleInfo
	selectedRole string
	fieldErrors  map[string]string
}

func (c *userShowController) computePerms(ctx context.Context, p *Page, targetID string) userShowPerms {
	isSelf := rbac.IsSelf(actorID(p), targetID)
	access := p.Deps.Users.Access()
	return userShowPerms{
		CanReadUser:         isSelf || access.CanReadUser(ctx, actorRole(p), actorID(p), targetID) == nil,
		CanChangeRole:       access.CanChangeRole(ctx, actorRole(p)) == nil,
		CanActivate:         access.CanActivate(ctx, actorRole(p)) == nil,
		CanChangeProfileAny: access.CanChangeProfileAny(ctx, actorRole(p), actorID(p), targetID) == nil,
		CanDelete:           access.CanDelete(ctx, actorRole(p)) == nil,
		CanRestore:          access.CanRestore(ctx, actorRole(p)) == nil,
	}
}

func (c *userShowController) reload(ctx context.Context, p *Page) error {
	id := paramID(p)
	if id == "" {
		return errors.New("kullanıcı kimliği gerekli")
	}
	if p.Deps.Users == nil {
		return errors.New("kullanıcı getirme servisi yapılandırılmamış")
	}
	view, err := p.Deps.Users.Get(ctx, appuser.GetQuery{
		UserID: id, ActorID: actorID(p), ActorRole: actorRole(p),
	})
	if err != nil {
		return err
	}
	c.profile = view
	c.perms = c.computePerms(ctx, p, view.ID)
	c.perms.CanActivate = c.perms.CanActivate && !view.Active && !view.Deleted
	c.perms.CanDelete = c.perms.CanDelete && !view.Deleted
	c.perms.CanRestore = c.perms.CanRestore && view.Deleted
	if c.perms.CanChangeRole && p.Deps.Authz != nil {
		roles, err := p.Deps.Authz.ListRoles(ctx)
		if err != nil {
			return err
		}
		c.roles = roles
		if c.selectedRole == "" {
			c.selectedRole = view.Role
		}
	}
	return nil
}

func (c *userShowController) Mount(ctx context.Context, p *Page) error {
	if err := c.reload(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
	}
	return nil
}

func (c *userShowController) Render(p *Page) (string, error) {
	phone := c.profile.Phone
	if phone == "" {
		phone = "—"
	}
	status := "Pasif"
	if c.profile.Deleted {
		status = "Silindi"
	} else if c.profile.Active {
		status = "Aktif"
	}
	mfaLabel := "Kapalı"
	if c.profile.MFAEnabled {
		mfaLabel = "Etkin"
	}
	emailLabel := "Bekliyor"
	if c.profile.EmailVerified {
		emailLabel = "Doğrulandı"
	}
	showManage := c.perms.CanChangeRole || c.perms.CanActivate || c.perms.CanChangeProfileAny || c.perms.CanDelete || c.perms.CanRestore

	roleOpts := make([]ViewOption, 0, len(c.roles))
	for _, r := range c.roles {
		roleOpts = append(roleOpts, ViewOption{Value: r.Name, Label: r.Name, Selected: r.Name == c.selectedRole})
	}

	data := map[string]any{
		"Name":                c.profile.Name,
		"Email":               c.profile.Email,
		"Phone":               phone,
		"Role":                c.profile.Role,
		"Status":              status,
		"EmailVerified":       c.profile.EmailVerified,
		"EmailVerifiedLabel":  emailLabel,
		"MFAEnabled":          c.profile.MFAEnabled,
		"MFALabel":            mfaLabel,
		"CreatedAt":           formatShort(c.profile.CreatedAt),
		"ShowManage":          showManage,
		"Deleted":             c.profile.Deleted,
		"DeletedAt":           formatShortPtr(c.profile.DeletedAt),
		"CanRestore":          c.perms.CanRestore,
		"CanActivate":         c.perms.CanActivate,
		"CanChangeRole":       c.perms.CanChangeRole,
		"CanChangeProfileAny": c.perms.CanChangeProfileAny,
		"CanDelete":           c.perms.CanDelete,
		"RoleOptions":         roleOpts,
		"ProfileName":         c.profile.Name,
		"ProfileEmail":        c.profile.Email,
		"ProfilePhone":        c.profile.Phone,
		"ErrRole":             viewFieldError(c.fieldErrors, "role"),
		"ErrName":             viewFieldError(c.fieldErrors, "name"),
		"ErrEmail":            viewFieldError(c.fieldErrors, "email"),
		"ErrPhone":            viewFieldError(c.fieldErrors, "phone"),
	}
	return p.RenderView("pages.user_show", data)
}

func (c *userShowController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	c.fieldErrors = nil
	id := paramID(p)
	if id == "" {
		p.Error = "kullanıcı kimliği gerekli"
		return nil
	}
	access := p.Deps.Users.Access()

	switch event {
	case "user.change_role":
		if err := requireAccess(access.CanChangeRole(ctx, actorRole(p))); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		req := userChangeRoleForm{Role: payloadString(payload, "role")}
		c.selectedRole = req.Role
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			_ = c.reload(ctx, p)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("rol güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeRole(ctx, appuser.ChangeRoleCommand{UserID: id, NewRole: req.Role}); err != nil {
			p.Error = accountDisplayErr(err)
			_ = c.reload(ctx, p)
			return nil
		}
		c.selectedRole = req.Role
		_ = c.reload(ctx, p)
		p.Notice = "kullanıcı rolü güncellendi"

	case "user.change_email":
		if err := requireAccess(access.CanChangeProfileAny(ctx, actorRole(p), actorID(p), id)); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		req := userChangeEmailForm{Email: payloadString(payload, "email")}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("e-posta güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeEmail(ctx, appuser.ChangeEmailCommand{
			UserID: id, NewEmail: req.Email, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.reload(ctx, p)
		p.Notice = "e-posta adresi güncellendi"

	case "user.change_name":
		if err := requireAccess(access.CanChangeProfileAny(ctx, actorRole(p), actorID(p), id)); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		req := userChangeNameForm{Name: payloadString(payload, "name")}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("ad güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeName(ctx, appuser.ChangeNameCommand{
			UserID: id, Name: req.Name, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.reload(ctx, p)
		p.Notice = "ad güncellendi"

	case "user.change_phone":
		if err := requireAccess(access.CanChangeProfileAny(ctx, actorRole(p), actorID(p), id)); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		req := userChangePhoneForm{Phone: payloadString(payload, "phone")}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("telefon güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangePhone(ctx, appuser.ChangePhoneCommand{
			UserID: id, Phone: req.Phone, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.reload(ctx, p)
		p.Notice = "telefon numarası güncellendi"

	case "user.activate":
		if err := requireAccess(access.CanActivate(ctx, actorRole(p))); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("aktifleştirme servisi yapılandırılmamış")
		}
		if err := p.Deps.Users.Activate(ctx, appuser.ActivateCommand{UserID: id}); err != nil {
			p.Error = accountDisplayErr(err)
			_ = c.reload(ctx, p)
			return nil
		}
		_ = c.reload(ctx, p)
		p.Notice = "kullanıcı aktifleştirildi"

	case "user.delete":
		if err := requireAccess(access.CanDelete(ctx, actorRole(p))); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("silme servisi yapılandırılmamış")
		}
		if err := p.Deps.Users.Delete(ctx, appuser.DeleteCommand{UserID: id}); err != nil {
			p.Error = accountDisplayErr(err)
			_ = c.reload(ctx, p)
			return nil
		}
		p.Notice = "kullanıcı silindi"
		p.Redirect = "/dashboard/users"

	case "user.restore":
		if err := requireAccess(access.CanRestore(ctx, actorRole(p))); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("geri yükleme servisi yapılandırılmamış")
		}
		if err := p.Deps.Users.Restore(ctx, appuser.RestoreCommand{UserID: id}); err != nil {
			p.Error = accountDisplayErr(err)
			_ = c.reload(ctx, p)
			return nil
		}
		_ = c.reload(ctx, p)
		p.Notice = "kullanıcı geri yüklendi"
	}
	return nil
}
