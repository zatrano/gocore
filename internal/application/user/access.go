package user

import (
	"context"

	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/rbac"
)

// ErrAccessDenied, çağıranın istenen işlem için yetkisi olmadığını belirtir.
var ErrAccessDenied = shared.NewDomainError(
	shared.KindForbidden, "auth.forbidden", "bu işlem için yetkiniz yok")

// Access, rol ve sahiplik tabanlı yetki kontrollerini merkezileştirir. Yetki
// çözümlemesi DİNAMİKTİR: rbac.Checker (veritabanı destekli resolver) üzerinden
// yapılır, böylece rol→izin değişiklikleri anında etkin olur.
type Access struct {
	checker rbac.Checker
}

// NewAccess, Access'i bir rbac.Checker ile kurar.
func NewAccess(checker rbac.Checker) Access {
	return Access{checker: checker}
}

// CanListUsers, kullanıcı listeleme iznini kontrol eder.
func (a Access) CanListUsers(ctx context.Context, role string) error {
	return a.require(ctx, role, rbac.PermUsersList)
}

// CanReadUser, profil okuma iznini kontrol eder (kendi profili veya users:read).
func (a Access) CanReadUser(ctx context.Context, role, actorID, targetID string) error {
	if rbac.IsSelf(actorID, targetID) {
		return nil
	}
	return a.require(ctx, role, rbac.PermUsersRead)
}

// CanChangeProfileAny, profil alanlarını düzenleme iznini kontrol eder (kendi hesabı veya
// users:email:change:any).
func (a Access) CanChangeProfileAny(ctx context.Context, role, actorID, targetID string) error {
	if rbac.IsSelf(actorID, targetID) {
		return nil
	}
	return a.require(ctx, role, rbac.PermUsersEmailAny)
}

// CanChangeEmail, e-posta değiştirme iznini kontrol eder (kendi hesabı veya
// users:email:change:any).
func (a Access) CanChangeEmail(ctx context.Context, role, actorID, targetID string) error {
	return a.CanChangeProfileAny(ctx, role, actorID, targetID)
}

// CanChangeRole, rol değiştirme iznini kontrol eder.
func (a Access) CanChangeRole(ctx context.Context, role string) error {
	return a.require(ctx, role, rbac.PermUsersRoleChange)
}

// CanActivate, kullanıcı aktifleştirme iznini kontrol eder.
func (a Access) CanActivate(ctx context.Context, role string) error {
	return a.require(ctx, role, rbac.PermUsersActivate)
}

// CanDelete, kullanıcı silme iznini kontrol eder.
func (a Access) CanDelete(ctx context.Context, role string) error {
	return a.require(ctx, role, rbac.PermUsersDelete)
}

// CanRestore, silinmiş kullanıcıyı geri yükleme iznini kontrol eder.
func (a Access) CanRestore(ctx context.Context, role string) error {
	return a.require(ctx, role, rbac.PermUsersRestore)
}

// PermissionsFor, rolün sahip olduğu izin listesini döner.
func (a Access) PermissionsFor(ctx context.Context, role string) ([]string, error) {
	return a.checker.PermissionsFor(ctx, role)
}

// require, verilen iznin varlığını doğrular; çözümleme hatasında güvenli tarafta
// kalınır (hata yayılır → HTTP katmanı 500 döner, sessizce izin verilmez).
func (a Access) require(ctx context.Context, role string, perm rbac.Permission) error {
	ok, err := a.checker.Allows(ctx, role, perm)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAccessDenied
	}
	return nil
}
