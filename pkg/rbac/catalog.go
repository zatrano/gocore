package rbac

// PermissionDef, izin kataloğu girdisidir (seed + açılış senkronu).
type PermissionDef struct {
	Permission  Permission
	Description string
}

// Catalog, kodda tanımlı tüm izinleri döner.
//
// Tek kaynak: bu liste. Açılışta authz.Syncer DB'ye eksik izinleri yazar.
// migrations/000002_seed aynı başlangıç izinlerini içerir (boş DB bootstrap);
// yeni izin eklerken Catalog'u güncellemek yeterlidir (Syncer admin'e atar).
// Şema değişiklikleri için 000001_schema'yı güncelleyin; parçalı migration eklemeyin
// (geliştirme aşamasında DB yeniden kurulur).
func Catalog() []PermissionDef {
	return []PermissionDef{
		{PermUsersList, "Kullanıcıları listeleme"},
		{PermUsersRead, "Başka kullanıcıların profilini görüntüleme"},
		{PermUsersActivate, "Kullanıcı hesabı aktifleştirme"},
		{PermUsersDelete, "Kullanıcı silme (soft delete)"},
		{PermUsersRestore, "Silinmiş kullanıcıyı geri getirme"},
		{PermUsersRoleChange, "Kullanıcı rolü değiştirme"},
		{PermUsersEmailAny, "Herhangi bir kullanıcının e-postasını değiştirme"},
		{PermUploadsCreate, "Dosya yükleme"},
		{PermRBACManage, "Rol ve izinleri yönetme"},
		{PermNotificationsSend, "Elle ve toplu bildirim/SMS/e-posta gönderme"},
		{PermNotificationsSettings, "SMS sağlayıcı ve bildirim ayarlarını yönetme"},
		{PermPaymentsCharge, "3DS tahsilat (BIN, başlatma, tamamlama)"},
		{PermPaymentsList, "Ödeme listesi ve detay görüntüleme"},
		{PermAuditList, "Denetim kayıtlarını listeleme"},
		{PermContactsList, "İletişim mesajlarını listeleme"},
	}
}
