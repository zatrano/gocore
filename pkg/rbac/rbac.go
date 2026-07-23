// Package rbac, rol tabanlı erişim kontrolü için izin adı sözleşmesini tanımlar.
//
// Bu paket bir domain modülü değildir. Rol/izin aggregate'leri
// internal/domain/rbac ve use-case'ler internal/application/authz altındadır.
//
// Burada yalnızca şunlar vardır:
//   - Perm* sabitleri (route/middleware guard'ların paylaştığı izin adları)
//   - Checker arayüzü (çözümleme sözleşmesi; DB destekli implementasyon authz'te)
//   - Catalog (açılışta DB senkronu için izin listesi)
//
// İzin kayıtları, roller ve rol→izin matrisi veritabanında tutulur.
package rbac

import "context"

// Permission, sistemdeki atomik bir yetki tanımlayıcısıdır (ör. "users:list").
type Permission string

// Route guard sabitleri — izin kaydı DB'de olmalı (migration veya POST /rbac/permissions).
const (
	PermUsersList             Permission = "users:list"
	PermUsersRead             Permission = "users:read"
	PermUsersActivate         Permission = "users:activate"
	PermUsersDelete           Permission = "users:delete"
	PermUsersRestore          Permission = "users:restore"
	PermUsersRoleChange       Permission = "users:role:change"
	PermUsersEmailAny         Permission = "users:email:change:any"
	PermUploadsCreate         Permission = "uploads:create"
	PermRBACManage            Permission = "rbac:manage"
	PermNotificationsSend     Permission = "notifications:send"
	PermNotificationsSettings Permission = "notifications:settings"
	PermPaymentsCharge        Permission = "payments:charge"
	PermPaymentsList          Permission = "payments:list"
	PermAuditList             Permission = "audit:list"
	PermContactsList          Permission = "contacts:list"
)

// Checker, bir rolün belirli bir izne sahip olup olmadığını çalışma zamanında
// çözer. Uygulamada veritabanı destekli (önbellekli) resolver bu arayüzü
// gerçekler; testlerde StaticChecker kullanılır. Hata dönerse çağıran taraf
// güvenli tarafta kalıp erişimi reddetmelidir (fail-closed).
type Checker interface {
	Allows(ctx context.Context, role string, perm Permission) (bool, error)
	PermissionsFor(ctx context.Context, role string) ([]string, error)
}

// IsSelf, hedef kaynağın çağıran kullanıcıya ait olup olmadığını kontrol eder.
func IsSelf(actorID, targetID string) bool {
	return actorID != "" && actorID == targetID
}
