package rbac

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	ErrRoleNotFound = shared.NewDomainError(
		shared.KindNotFound, "rbac.role_not_found", "rol bulunamadı")

	ErrRoleExists = shared.NewDomainError(
		shared.KindConflict, "rbac.role_exists", "bu adda bir rol zaten mevcut")

	ErrSystemRoleImmutable = shared.NewDomainError(
		shared.KindConflict, "rbac.system_role", "sistem rolleri değiştirilemez")

	ErrRoleInUse = shared.NewDomainError(
		shared.KindConflict, "rbac.role_in_use", "bu rol kullanıcılara atanmış olduğu için silinemez")

	ErrInvalidRoleName = shared.NewDomainError(
		shared.KindValidation, "rbac.invalid_role_name", "rol adı geçersiz (küçük harf, rakam, - ve _; 2-32 karakter)")

	ErrInvalidPermissionName = shared.NewDomainError(
		shared.KindValidation, "rbac.invalid_permission_name", "izin adı geçersiz")

	ErrUnknownPermission = shared.NewDomainError(
		shared.KindValidation, "rbac.unknown_permission", "bilinmeyen izin")
	ErrPermissionExists = shared.NewDomainError(
		shared.KindConflict, "rbac.permission_exists", "izin zaten mevcut")
	ErrPermissionNotFound = shared.NewDomainError(
		shared.KindNotFound, "rbac.permission_not_found", "izin bulunamadı")
)
