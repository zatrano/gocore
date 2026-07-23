package rbac

import (
	"strings"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// Role, RBAC bounded context'inin aggregate root'udur. Rol adı, açıklaması,
// sistem rolü bayrağı ve atanmış izinleri bu tip üzerinden yönetilir.
type Role struct {
	shared.EventRecorder

	name        RoleName
	description string
	isSystem    bool
	permissions []PermissionName
}

// Permission, izin kataloğundaki bir kaydı temsil eder (salt okunur katalog girdisi).
type Permission struct {
	name        PermissionName
	description string
}

// NewPermission, katalog girdisini oluşturur.
func NewPermission(name, description string) (Permission, error) {
	pn, err := ParsePermissionName(name)
	if err != nil {
		return Permission{}, err
	}
	return Permission{name: pn, description: strings.TrimSpace(description)}, nil
}

// RehydratePermission, persistence katmanından okunan izin kaydını yeniden oluşturur.
func RehydratePermission(name PermissionName, description string) Permission {
	return Permission{name: name, description: description}
}

func (p Permission) Name() PermissionName { return p.name }
func (p Permission) Description() string  { return p.description }

// CreateRole, yeni bir özel (sistem-olmayan) rol oluşturur.
func CreateRole(name, description string) (*Role, error) {
	rn, err := ParseRoleName(strings.TrimSpace(name))
	if err != nil {
		return nil, err
	}
	desc := strings.TrimSpace(description)
	r := &Role{
		name:        rn,
		description: desc,
		isSystem:    false,
	}
	r.Record(RoleCreatedEvent{
		BaseEvent:   shared.NewBaseEvent(EventRoleCreated, rn.String()),
		Name:        rn.String(),
		Description: desc,
		IsSystem:    false,
	})
	return r, nil
}

// CreateSystemRole, açılış senkronu için sistem rolü oluşturur.
func CreateSystemRole(name RoleName, description string) (*Role, error) {
	if _, err := ParseRoleName(name.String()); err != nil {
		return nil, err
	}
	desc := strings.TrimSpace(description)
	return &Role{
		name:        name,
		description: desc,
		isSystem:    true,
	}, nil
}

// Rehydrate, persistence katmanından okunan rolü yeniden oluşturur (event üretmez).
func Rehydrate(name RoleName, description string, isSystem bool, permissions []PermissionName) *Role {
	perms := append([]PermissionName(nil), permissions...)
	return &Role{
		name:        name,
		description: strings.TrimSpace(description),
		isSystem:    isSystem,
		permissions: perms,
	}
}

// UpdateDescription, rol açıklamasını günceller.
func (r *Role) UpdateDescription(description string) {
	desc := strings.TrimSpace(description)
	if r.description == desc {
		return
	}
	r.description = desc
	r.Record(RoleDescriptionChangedEvent{
		BaseEvent:   shared.NewBaseEvent(EventRoleDescriptionChanged, r.name.String()),
		Description: desc,
	})
}

// ReplacePermissions, rolün izin kümesini tamamen değiştirir.
func (r *Role) ReplacePermissions(perms []PermissionName) {
	r.permissions = append(r.permissions[:0], perms...)
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = p.String()
	}
	r.Record(PermissionsReplacedEvent{
		BaseEvent:   shared.NewBaseEvent(EventPermissionsReplaced, r.name.String()),
		Permissions: names,
	})
}

// EnsureDeletable, rolün silinip silinemeyeceğini doğrular.
func (r *Role) EnsureDeletable(assignedUserCount int64) error {
	if r.isSystem {
		return ErrSystemRoleImmutable
	}
	if assignedUserCount > 0 {
		return ErrRoleInUse
	}
	return nil
}

// MarkDeleted, silme olayını kaydeder (kalıcılık repository tarafından yapılır).
func (r *Role) MarkDeleted() {
	r.Record(RoleDeletedEvent{
		BaseEvent: shared.NewBaseEvent(EventRoleDeleted, r.name.String()),
		Name:      r.name.String(),
	})
}

func (r *Role) Name() RoleName                { return r.name }
func (r *Role) Description() string           { return r.description }
func (r *Role) IsSystem() bool                { return r.isSystem }
func (r *Role) Permissions() []PermissionName { return append([]PermissionName(nil), r.permissions...) }
func (r *Role) PermissionNames() []string {
	out := make([]string, len(r.permissions))
	for i, p := range r.permissions {
		out[i] = p.String()
	}
	return out
}
