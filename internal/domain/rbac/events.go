package rbac

import "github.com/zatrano/gocore/internal/domain/shared"

const (
	EventRoleCreated            = "rbac.role_created"
	EventRoleDescriptionChanged = "rbac.role_description_changed"
	EventPermissionsReplaced    = "rbac.permissions_replaced"
	EventRoleDeleted            = "rbac.role_deleted"
)

// RoleCreatedEvent, yeni bir rol tanımlandığında üretilir.
type RoleCreatedEvent struct {
	shared.BaseEvent
	Name        string
	Description string
	IsSystem    bool
}

// RoleDescriptionChangedEvent, rol açıklaması güncellendiğinde üretilir.
type RoleDescriptionChangedEvent struct {
	shared.BaseEvent
	Description string
}

// PermissionsReplacedEvent, rolün izin kümesi değiştirildiğinde üretilir.
type PermissionsReplacedEvent struct {
	shared.BaseEvent
	Permissions []string
}

// RoleDeletedEvent, bir rol silindiğinde üretilir.
type RoleDeletedEvent struct {
	shared.BaseEvent
	Name string
}
