package user

import (
	"context"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

// --- Command girdileri (immutable DTO) ---

// RegisterCommand, yeni kullanıcı kaydı için girdidir.
type RegisterCommand struct {
	Email           string
	Name            string
	Password        string
	Role            string
	Phone           string // opsiyonel; boşsa kaydedilmez
	PreferredLocale string // boşsa politika varsayılanını kullanır
	// AllowPrivilegedRole, true ise Role alanı admin dahil kabul edilir (seed/dahili).
	// Public API her zaman false gönderir → yalnızca user rolü atanır.
	AllowPrivilegedRole bool
}

// ChangeLocaleCommand, kullanıcının kalıcı dil tercihini günceller.
type ChangeLocaleCommand struct {
	UserID string
	Locale string
}

// ActivateCommand, kullanıcı aktifleştirme girdisidir.
type ActivateCommand struct {
	UserID string
}

// ChangeEmailCommand, e-posta değiştirme girdisidir.
type ChangeEmailCommand struct {
	UserID    string
	NewEmail  string
	ActorID   string
	ActorRole string
}

// ChangePhoneCommand, telefon numarası değiştirme/kaldırma girdisidir.
type ChangePhoneCommand struct {
	UserID    string
	Phone     string // boş = telefonu kaldır
	ActorID   string
	ActorRole string
}

// ChangeNameCommand, görünen ad değiştirme girdisidir.
type ChangeNameCommand struct {
	UserID    string
	Name      string
	ActorID   string
	ActorRole string
}

// ChangeRoleCommand, kullanıcı rolü değiştirme girdisidir (admin).
type ChangeRoleCommand struct {
	UserID  string
	NewRole string
}

// DeleteCommand, kullanıcıyı yazılımsal silme girdisidir.
type DeleteCommand struct {
	UserID string
}

// RestoreCommand, yazılımsal silmeyi geri alma girdisidir.
type RestoreCommand struct {
	UserID string
}

// RegisterHandler, kullanıcı kaydı command'ini işler. Bağımlılıkları %100
// constructor injection ile alır; hiçbir global state kullanmaz.
type RegisterHandler struct {
	repo      user.Repository
	hasher    appshared.PasswordHasher
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	locales   LocalePolicy
	roles     RoleChecker
}

// NewRegisterHandler, RegisterHandler'ı bağımlılıklarıyla kurar. roles, yalnızca
// ayrıcalıklı kayıtta (AllowPrivilegedRole) rolün varlığını doğrulamak için
// kullanılır; public kayıt her zaman "user" rolünü atar.
func NewRegisterHandler(
	repo user.Repository,
	hasher appshared.PasswordHasher,
	publisher appshared.EventPublisher,
	tx appshared.TxManager,
	locales LocalePolicy,
	roles RoleChecker,
) *RegisterHandler {
	return &RegisterHandler{repo: repo, hasher: hasher, publisher: publisher, tx: tx, locales: locales, roles: roles}
}

// Handle, yeni bir kullanıcı oluşturur. Tüm adımlar tek transaction içinde
// yürütülür; domain event'leri aynı transaction içinde yayınlanır.
func (h *RegisterHandler) Handle(ctx context.Context, cmd RegisterCommand) (View, error) {
	email, err := user.NewEmail(cmd.Email)
	if err != nil {
		return View{}, err
	}

	role := user.RoleUser
	if cmd.AllowPrivilegedRole {
		if cmd.Role == "" {
			return View{}, user.ErrInvalidRole
		}
		parsed, err := user.ParseRole(cmd.Role)
		if err != nil {
			return View{}, err
		}
		// Rol dinamik olduğundan varlığını da doğrula.
		exists, err := h.roles.RoleExists(ctx, cmd.Role)
		if err != nil {
			return View{}, err
		}
		if !exists {
			return View{}, user.ErrInvalidRole
		}
		role = parsed
	}

	encoded, err := h.hasher.Hash(ctx, cmd.Password)
	if err != nil {
		return View{}, err
	}
	hashed, err := user.NewHashedPassword(encoded)
	if err != nil {
		return View{}, err
	}

	locale, err := h.locales.Resolve(cmd.PreferredLocale)
	if err != nil {
		return View{}, err
	}
	phone, err := user.NewPhone(cmd.Phone)
	if err != nil {
		return View{}, err
	}

	u, err := user.Register(email, cmd.Name, hashed, role, locale, phone)
	if err != nil {
		return View{}, err
	}

	// Benzersizlik kontrolü + kayıt atomik olsun diye transaction kullanılır.
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		exists, err := h.repo.ExistsByEmail(ctx, email)
		if err != nil {
			return err
		}
		if exists {
			return user.ErrEmailAlreadyExists
		}
		if err := h.repo.Save(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// ActivateHandler, kullanıcı aktifleştirme command'ini işler.
type ActivateHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
}

// NewActivateHandler, ActivateHandler'ı kurar.
func NewActivateHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager) *ActivateHandler {
	return &ActivateHandler{repo: repo, publisher: publisher, tx: tx}
}

// Handle, kullanıcıyı aktifleştirir.
func (h *ActivateHandler) Handle(ctx context.Context, cmd ActivateCommand) error {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := u.Activate(); err != nil {
			return err
		}
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	return err
}

// ChangeEmailHandler, e-posta değiştirme command'ini işler.
type ChangeEmailHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	access    Access
}

// NewChangeEmailHandler, ChangeEmailHandler'ı kurar.
func NewChangeEmailHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager, access Access) *ChangeEmailHandler {
	return &ChangeEmailHandler{repo: repo, publisher: publisher, tx: tx, access: access}
}

// Handle, kullanıcının e-postasını değiştirir; yeni e-posta başkası tarafından
// kullanılıyorsa çakışma hatası döner. Kendi hesabı veya users:email:change:any izni gerekir.
func (h *ChangeEmailHandler) Handle(ctx context.Context, cmd ChangeEmailCommand) (View, error) {
	if err := h.access.CanChangeEmail(ctx, cmd.ActorRole, cmd.ActorID, cmd.UserID); err != nil {
		return View{}, err
	}
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return View{}, err
	}
	newEmail, err := user.NewEmail(cmd.NewEmail)
	if err != nil {
		return View{}, err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		exists, err := h.repo.ExistsByEmail(ctx, newEmail)
		if err != nil {
			return err
		}
		if exists {
			return user.ErrEmailAlreadyExists
		}
		u.ChangeEmail(newEmail)
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// ChangePhoneHandler, telefon numarası değiştirme command'ini işler.
type ChangePhoneHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	access    Access
}

// NewChangePhoneHandler, ChangePhoneHandler'ı kurar.
func NewChangePhoneHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager, access Access) *ChangePhoneHandler {
	return &ChangePhoneHandler{repo: repo, publisher: publisher, tx: tx, access: access}
}

// Handle, kullanıcının telefonunu günceller veya kaldırır. Kendi hesabı veya
// users:email:change:any izni (profil yönetimi) gerekir.
func (h *ChangePhoneHandler) Handle(ctx context.Context, cmd ChangePhoneCommand) (View, error) {
	if err := h.access.CanChangeEmail(ctx, cmd.ActorRole, cmd.ActorID, cmd.UserID); err != nil {
		return View{}, err
	}
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return View{}, err
	}
	phone, err := user.NewPhone(cmd.Phone)
	if err != nil {
		return View{}, err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		u.ChangePhone(phone)
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// ChangeNameHandler, görünen ad değiştirme command'ini işler.
type ChangeNameHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	access    Access
}

// NewChangeNameHandler, ChangeNameHandler'ı kurar.
func NewChangeNameHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager, access Access) *ChangeNameHandler {
	return &ChangeNameHandler{repo: repo, publisher: publisher, tx: tx, access: access}
}

// Handle, kullanıcının görünen adını günceller. Kendi hesabı veya
// users:email:change:any izni (profil yönetimi) gerekir.
func (h *ChangeNameHandler) Handle(ctx context.Context, cmd ChangeNameCommand) (View, error) {
	if err := h.access.CanChangeEmail(ctx, cmd.ActorRole, cmd.ActorID, cmd.UserID); err != nil {
		return View{}, err
	}
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return View{}, err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := u.Rename(cmd.Name); err != nil {
			return err
		}
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// DeleteHandler, kullanıcıyı yazılımsal olarak silme (soft delete) command'ini
// işler: aggregate yüklenir, domain invariant'ı doğrulanır (zaten silinmiş mi?),
// kalıcılaştırılır ve DeletedEvent yayınlanır.
type DeleteHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
}

// NewDeleteHandler, DeleteHandler'ı kurar.
func NewDeleteHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager) *DeleteHandler {
	return &DeleteHandler{repo: repo, publisher: publisher, tx: tx}
}

// Handle, kullanıcıyı yazılımsal olarak siler.
func (h *DeleteHandler) Handle(ctx context.Context, cmd DeleteCommand) error {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		// Domain davranışı: invariant + DeletedEvent üretimi.
		if err := u.Delete(); err != nil {
			return err
		}
		if err := h.repo.Delete(ctx, id); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	return err
}

// RestoreHandler, yazılımsal silmeyi geri alma command'ini işler.
type RestoreHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
}

// NewRestoreHandler, RestoreHandler'ı kurar.
func NewRestoreHandler(repo user.Repository, publisher appshared.EventPublisher, tx appshared.TxManager) *RestoreHandler {
	return &RestoreHandler{repo: repo, publisher: publisher, tx: tx}
}

// Handle, yazılımsal silmeyi geri alır ve RestoredEvent yayınlar.
func (h *RestoreHandler) Handle(ctx context.Context, cmd RestoreCommand) error {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return err
	}

	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		u, err := h.repo.FindByIDIncludeDeleted(ctx, id)
		if err != nil {
			return err
		}
		if !u.IsDeleted() {
			return user.ErrNotDeleted
		}
		if err := h.repo.Restore(ctx, id); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, user.NewRestoredEvent(id.String()))
	})
	return err
}

// ChangeLocaleHandler, kullanıcının kalıcı dil tercihini günceller.
type ChangeLocaleHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	locales   LocalePolicy
}

// NewChangeLocaleHandler, ChangeLocaleHandler'ı kurar.
func NewChangeLocaleHandler(
	repo user.Repository,
	publisher appshared.EventPublisher,
	tx appshared.TxManager,
	locales LocalePolicy,
) *ChangeLocaleHandler {
	return &ChangeLocaleHandler{repo: repo, publisher: publisher, tx: tx, locales: locales}
}

// Handle, preferred_locale alanını günceller.
func (h *ChangeLocaleHandler) Handle(ctx context.Context, cmd ChangeLocaleCommand) (View, error) {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return View{}, err
	}
	locale, err := h.locales.Resolve(cmd.Locale)
	if err != nil {
		return View{}, err
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		u.ChangePreferredLocale(locale)
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return newView(u), nil
}

// ChangeRoleHandler, kullanıcı rolü değiştirme command'ini işler. Son admin
// koruması uygulanır; rol değişince hedef kullanıcının tüm oturumları iptal edilir.
type ChangeRoleHandler struct {
	repo      user.Repository
	publisher appshared.EventPublisher
	sessions  appshared.SessionRevoker
	tx        appshared.TxManager
	roles     RoleChecker
}

// NewChangeRoleHandler, ChangeRoleHandler'ı kurar.
func NewChangeRoleHandler(
	repo user.Repository,
	publisher appshared.EventPublisher,
	sessions appshared.SessionRevoker,
	tx appshared.TxManager,
	roles RoleChecker,
) *ChangeRoleHandler {
	return &ChangeRoleHandler{repo: repo, publisher: publisher, sessions: sessions, tx: tx, roles: roles}
}

// Handle, kullanıcının rolünü günceller.
func (h *ChangeRoleHandler) Handle(ctx context.Context, cmd ChangeRoleCommand) (View, error) {
	id, err := user.ParseID(cmd.UserID)
	if err != nil {
		return View{}, err
	}
	newRole, err := user.ParseRole(cmd.NewRole)
	if err != nil {
		return View{}, err
	}
	// Hedef rol dinamik olduğundan varlığını doğrula.
	exists, err := h.roles.RoleExists(ctx, cmd.NewRole)
	if err != nil {
		return View{}, err
	}
	if !exists {
		return View{}, user.ErrInvalidRole
	}

	var u *user.User
	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		u, err = h.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		// Son admin koruması: tek kalan admin başka role düşürülemez.
		if u.Role() == user.RoleAdmin && newRole != user.RoleAdmin {
			count, err := h.repo.CountActiveByRole(ctx, user.RoleAdmin)
			if err != nil {
				return err
			}
			if count <= 1 {
				return user.ErrCannotDemoteLastAdmin
			}
		}
		if err := u.ChangeRole(newRole); err != nil {
			return err
		}
		if err := h.repo.Update(ctx, u); err != nil {
			return err
		}
		return h.publisher.Publish(ctx, u.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}

	_ = h.sessions.RevokeAll(ctx, u.ID().String())
	return newView(u), nil
}
