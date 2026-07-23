package rbac

import "context"

// Repository, Role aggregate'i ve izin kayıtları için persistence portudur.
// Domain katmanı yalnızca bu arayüzü bilir; PostgreSQL implementasyonu
// adapters katmanındadır.
type Repository interface {
	// Save, yeni bir rolü kalıcılaştırır.
	Save(ctx context.Context, role *Role) error

	// UpdateDescription, rol açıklamasını günceller. Yoksa ErrRoleNotFound.
	UpdateDescription(ctx context.Context, name RoleName, description string) error

	// ReplacePermissions, rolün izin kümesini tamamen değiştirir.
	ReplacePermissions(ctx context.Context, name RoleName, perms []PermissionName) error

	// FindByName, tek rolü izinleriyle birlikte getirir. Yoksa ErrRoleNotFound.
	FindByName(ctx context.Context, name RoleName) (*Role, error)

	// List, tüm rolleri izinleriyle birlikte listeler.
	List(ctx context.Context) ([]*Role, error)

	// Delete, sistem-olmayan rolü siler. Yoksa ErrRoleNotFound.
	Delete(ctx context.Context, name RoleName) error

	// Exists, rolün tanımlı olup olmadığını döner.
	Exists(ctx context.Context, name RoleName) (bool, error)

	// CountUsersWithRole, role atanmış canlı kullanıcı sayısını döner.
	CountUsersWithRole(ctx context.Context, name RoleName) (int64, error)

	// GrantAllPermissions, DB'deki tüm izinleri role atar (admin senkronu).
	GrantAllPermissions(ctx context.Context, name RoleName) error

	// CreatePermission, yeni izin kaydı oluşturur.
	CreatePermission(ctx context.Context, perm Permission) error

	// PermissionExists, izin adının kayıtlı olup olmadığını döner.
	PermissionExists(ctx context.Context, name PermissionName) (bool, error)

	// UpdatePermissionDescription, izin açıklamasını günceller.
	UpdatePermissionDescription(ctx context.Context, name PermissionName, description string) error

	// ListPermissions, tüm izinleri döner.
	ListPermissions(ctx context.Context) ([]Permission, error)
}
