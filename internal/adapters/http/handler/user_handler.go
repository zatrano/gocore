package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/validation"
)

// UserHandler, kullanıcı HTTP uç noktalarını use-case'lere bağlar.
type UserHandler struct {
	users     *appuser.Service
	validate  *validator.Validate
	turnstile turnstile.Verifier
}

// UserDeps, UserHandler bağımlılıklarını gruplar.
type UserDeps struct {
	Users     *appuser.Service
	Validate  *validator.Validate
	Turnstile turnstile.Verifier
}

// NewUserHandler, handler'ı kurar.
func NewUserHandler(d UserDeps) *UserHandler {
	return &UserHandler{users: d.Users, validate: d.Validate, turnstile: d.Turnstile}
}

// --- Request DTO'ları (immutable girdi, doğrulama tag'leri ile) ---

type registerRequest struct {
	Email           string `json:"email" validate:"required,email" sanitize:"email"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	Password        string `json:"password" validate:"required,min=8,max=128"`
	Phone           string `json:"phone" validate:"omitempty,phone" sanitize:"phone"`
	PreferredLocale string `json:"preferred_locale"`
	TurnstileToken  string `json:"turnstile_token"`
}

type adminRegisterRequest struct {
	Email           string `json:"email" validate:"required,email" sanitize:"email"`
	Name            string `json:"name" validate:"required,min=2,max=100"`
	Password        string `json:"password" validate:"required,min=8,max=128"`
	Role            string `json:"role" validate:"required"`
	Phone           string `json:"phone" validate:"omitempty,phone" sanitize:"phone"`
	PreferredLocale string `json:"preferred_locale"`
}

type changeEmailRequest struct {
	Email string `json:"email" validate:"required,email" sanitize:"email"`
}

type changePhoneRequest struct {
	Phone string `json:"phone" validate:"omitempty,phone" sanitize:"phone"`
}

type changeNameRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

type changeLocaleRequest struct {
	Locale string `json:"locale" validate:"required"`
}

type changeRoleRequest struct {
	// Roller dinamik olduğundan burada sabit bir liste (oneof) zorlanmaz; rolün
	// geçerliliği use-case katmanında (RoleChecker) doğrulanır.
	Role string `json:"role" validate:"required"`
}

// Register, POST /users — yeni kullanıcı oluşturur (her zaman user rolü).
func (h *UserHandler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := adapters.VerifyTurnstile(c, h.turnstile, req.TurnstileToken); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	locale := req.PreferredLocale
	if locale == "" {
		locale = string(i18n.LocaleFromContext(c.Context()))
	}

	view, err := h.users.Register(c.Context(), appuser.RegisterCommand{
		Email:               req.Email,
		Name:                req.Name,
		Password:            req.Password,
		Phone:               req.Phone,
		PreferredLocale:     locale,
		AllowPrivilegedRole: false,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusCreated, "success.user.registered", "kullanıcı başarıyla kaydedildi", view)
}

// AdminCreate, POST /users/create — yönetici yeni kullanıcı oluşturur (rol seçilebilir).
func (h *UserHandler) AdminCreate(c fiber.Ctx) error {
	var req adminRegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	locale := req.PreferredLocale
	if locale == "" {
		locale = string(i18n.LocaleFromContext(c.Context()))
	}

	view, err := h.users.Register(c.Context(), appuser.RegisterCommand{
		Email:               req.Email,
		Name:                req.Name,
		Password:            req.Password,
		Role:                req.Role,
		Phone:               req.Phone,
		PreferredLocale:     locale,
		AllowPrivilegedRole: true,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusCreated, "success.user.registered", "kullanıcı başarıyla kaydedildi", view)
}

// Get, GET /users/:id — tek kullanıcı getirir (kendi profili veya users:read).
func (h *UserHandler) Get(c fiber.Ctx) error {
	actorID, actorRole := adapters.ActorFromCtx(c)
	view, err := h.users.Get(c.Context(), appuser.GetQuery{
		UserID:    c.Params("id"),
		ActorID:   actorID,
		ActorRole: actorRole,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}

// List, GET /users — filtreleme + sıralama + sayfalama (users:list).
func (h *UserHandler) List(c fiber.Ctx) error {
	_, actorRole := adapters.ActorFromCtx(c)
	cursor := c.Query("cursor")
	pageNum := adapters.ParsePage(c.Query("page"))
	if cursor != "" {
		pageNum = 1
	}
	q := appuser.ListQuery{
		ActorRole: actorRole,
		Role:      c.Query("role"),
		Search:    c.Query("search"),
		Deleted:   c.Query("deleted"),
		Page:      pageNum,
		Limit:     adapters.ParseLimit(c.Query("limit")),
		Ascending: c.Query("order") == "asc",
		Cursor:    cursor,
	}
	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		q.Active = &active
	}

	page, err := h.users.List(c.Context(), q)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, page)
}

// Activate, POST /users/:id/activate — kullanıcıyı aktifleştirir.
func (h *UserHandler) Activate(c fiber.Ctx) error {
	if err := h.users.Activate(c.Context(), appuser.ActivateCommand{UserID: c.Params("id")}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.user.activated", "kullanıcı aktifleştirildi")
}

// Delete, DELETE /users/:id — kullanıcıyı yazılımsal olarak siler (soft delete).
func (h *UserHandler) Delete(c fiber.Ctx) error {
	if err := h.users.Delete(c.Context(), appuser.DeleteCommand{UserID: c.Params("id")}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.user.deleted", "kullanıcı silindi")
}

// Restore, POST /users/:id/restore — yazılımsal silmeyi geri alır.
func (h *UserHandler) Restore(c fiber.Ctx) error {
	if err := h.users.Restore(c.Context(), appuser.RestoreCommand{UserID: c.Params("id")}); err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusOK, "success.user.restored", "kullanıcı geri yüklendi")
}

// ChangeEmail, PATCH /users/:id/email — e-posta değiştirir (kendi veya admin).
func (h *UserHandler) ChangeEmail(c fiber.Ctx) error {
	return h.changeEmailFor(c, c.Params("id"))
}

// ChangeMyEmail, PATCH /users/me/email — oturum açmış kullanıcının e-postasını günceller.
func (h *UserHandler) ChangeMyEmail(c fiber.Ctx) error {
	userID, _ := adapters.ActorFromCtx(c)
	return h.changeEmailFor(c, userID)
}

func (h *UserHandler) changeEmailFor(c fiber.Ctx, userID string) error {
	var req changeEmailRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	actorID, actorRole := adapters.ActorFromCtx(c)
	view, err := h.users.ChangeEmail(c.Context(), appuser.ChangeEmailCommand{
		UserID: userID, NewEmail: req.Email, ActorID: actorID, ActorRole: actorRole,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.user.email_changed", "e-posta adresi güncellendi", view)
}

// ChangePhone, PATCH /users/:id/phone — telefon değiştirir/kaldırır (kendi veya admin).
func (h *UserHandler) ChangePhone(c fiber.Ctx) error {
	return h.changePhoneFor(c, c.Params("id"))
}

// ChangeMyPhone, PATCH /users/me/phone — oturum açmış kullanıcının telefonunu günceller.
func (h *UserHandler) ChangeMyPhone(c fiber.Ctx) error {
	userID, _ := adapters.ActorFromCtx(c)
	return h.changePhoneFor(c, userID)
}

func (h *UserHandler) changePhoneFor(c fiber.Ctx, userID string) error {
	var req changePhoneRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	actorID, actorRole := adapters.ActorFromCtx(c)
	view, err := h.users.ChangePhone(c.Context(), appuser.ChangePhoneCommand{
		UserID: userID, Phone: req.Phone, ActorID: actorID, ActorRole: actorRole,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.user.phone_changed", "telefon numarası güncellendi", view)
}

// ChangeName, PATCH /users/:id/name — görünen adı değiştirir (kendi veya admin).
func (h *UserHandler) ChangeName(c fiber.Ctx) error {
	return h.changeNameFor(c, c.Params("id"))
}

// ChangeMyName, PATCH /users/me/name — oturum açmış kullanıcının adını günceller.
func (h *UserHandler) ChangeMyName(c fiber.Ctx) error {
	userID, _ := adapters.ActorFromCtx(c)
	return h.changeNameFor(c, userID)
}

func (h *UserHandler) changeNameFor(c fiber.Ctx, userID string) error {
	var req changeNameRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	actorID, actorRole := adapters.ActorFromCtx(c)
	view, err := h.users.ChangeName(c.Context(), appuser.ChangeNameCommand{
		UserID: userID, Name: req.Name, ActorID: actorID, ActorRole: actorRole,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.user.name_changed", "ad güncellendi", view)
}

// ChangeRole, PATCH /users/:id/role — kullanıcı rolünü değiştirir (admin).
func (h *UserHandler) ChangeRole(c fiber.Ctx) error {
	var req changeRoleRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	view, err := h.users.ChangeRole(c.Context(), appuser.ChangeRoleCommand{
		UserID:  c.Params("id"),
		NewRole: req.Role,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.user.role_changed", "kullanıcı rolü güncellendi", view)
}

// ChangeLocale, PATCH /users/me/locale — kalıcı dil tercihini günceller.
func (h *UserHandler) ChangeLocale(c fiber.Ctx) error {
	var req changeLocaleRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}

	userID, _ := adapters.ActorFromCtx(c)
	view, err := h.users.ChangeLocale(c.Context(), appuser.ChangeLocaleCommand{
		UserID: userID,
		Locale: req.Locale,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusOK, "success.user.locale_changed", "dil tercihi güncellendi", view)
}

// Me, GET /users/me — oturum açmış kullanıcının profili ve izinleri.
func (h *UserHandler) Me(c fiber.Ctx) error {
	userID, role := adapters.ActorFromCtx(c)
	view, err := h.users.Get(c.Context(), appuser.GetQuery{
		UserID: userID, ActorID: userID, ActorRole: role,
	})
	if err != nil {
		return render.Error(c, err)
	}
	perms, err := h.users.Access().PermissionsFor(c.Context(), role)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, appuser.MeView{
		View:        view,
		Permissions: perms,
	})
}
