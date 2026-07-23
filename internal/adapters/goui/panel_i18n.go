package goui

import (
	"html"
	"regexp"
	"sort"
	"strings"
)

var filesUploadedNotice = regexp.MustCompile(`^dosyalar yüklendi \((\d+)/(\d+)\)$`)

// translatePanelHTML, panel controller'ların ürettiği statik metinleri çevirir.
// Yalnızca HTML metin düğümleri ve attribute değerlerinde eşleşir (kısa kelime
// çakışmalarını önlemek için ham ReplaceAll kullanılmaz).
func (p *Page) translatePanelHTML(body string) string {
	keys := make([]string, 0, len(panelTextKeys))
	for fallback := range panelTextKeys {
		keys = append(keys, fallback)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, fallback := range keys {
		translated := html.EscapeString(p.T(panelTextKeys[fallback], fallback))
		if translated == fallback {
			continue
		}
		body = strings.ReplaceAll(body, ">"+fallback+"<", ">"+translated+"<")
		body = strings.ReplaceAll(body, `="`+fallback+`"`, `="`+translated+`"`)
		body = strings.ReplaceAll(body, `='`+fallback+`'`, `='`+translated+`'`)
	}
	return body
}

// translatePanelNotice, flash/notice metinlerini çevirir.
func (p *Page) translatePanelNotice(msg string) string {
	if msg == "" {
		return msg
	}
	if m := filesUploadedNotice.FindStringSubmatch(msg); len(m) == 3 {
		return p.T("dashboard.notice.files_uploaded", msg, m[1], m[2])
	}
	if key, ok := panelTextKeys[msg]; ok {
		return p.T(key, msg)
	}
	return msg
}

var panelTextKeys = map[string]string{
	"← Hesaba dön": "common.back_to_account", "← Listeye dön": "common.back_to_list",
	"← Kullanıcılara dön": "dashboard.users.back_to_list", "← Rollere dön": "dashboard.rbac.back_to_roles",
	"← Ödemelere dön": "dashboard.payments.back_to_list", "← Ödeme listesine dön": "dashboard.payments.back_to_list",
	"← SMS listesine dön": "dashboard.settings.back_to_sms", "← Toplu Gönderim": "dashboard.notifications.back_to_bulk",
	"← Tekil Gönderim": "dashboard.notifications.back_to_single", "← Liste ile Gönderim": "dashboard.notifications.back_to_list",
	"Bir veya daha fazla alan geçersiz": "common.invalid_fields", "Kayıt bulunamadı.": "common.no_records",
	"Filtreler": "common.filters", "Filtrele": "common.filter", "Temizle": "common.clear",
	"Detay": "common.details", "Düzenle": "common.edit", "Kaydet": "common.save",
	"Oluştur": "common.create", "Ekle": "common.add", "Gönder": "common.send",
	"Yükle": "common.upload", "Aktif": "common.active", "Pasif": "common.inactive",
	"Evet": "common.yes", "Hayır": "common.no", "Durum": "common.status",
	"Tarih": "common.date", "Ad": "common.name", "E-posta": "common.email",
	"Telefon": "common.phone", "Rol": "common.role", "Açıklama": "common.description",
	"İçerik": "common.content", "Başlık": "common.title", "Dil": "common.language",
	"Dashboard": "dashboard.title", "Hesabım": "dashboard.account.title",
	"İletişim Mesajları": "dashboard.contacts.title", "Mesaj Listesi": "dashboard.contacts.list",
	"İletişim Mesajı": "dashboard.contacts.detail", "Okunmamış": "dashboard.contacts.unread",
	"Okundu": "dashboard.contacts.read", "Okundu işaretle": "dashboard.contacts.mark_read",
	"Yeni Kullanıcı": "dashboard.users.new", "Kullanıcılar": "dashboard.users.title",
	"Kullanıcı Listesi": "dashboard.users.list", "Ad veya e-posta ara": "dashboard.users.search_placeholder",
	"Tüm roller": "dashboard.users.all_roles", "Kayıt durumu": "dashboard.users.record_status",
	"Canlı kayıtlar": "dashboard.users.live_records", "Yalnızca silinenler": "dashboard.users.deleted_only",
	"Silindi": "dashboard.users.deleted", "Bekliyor": "common.pending", "Doğrulandı": "common.verified",
	"Etkin": "common.enabled", "Kapalı": "common.disabled", "Yönetim": "dashboard.users.management",
	"Aktifleştir": "dashboard.users.activate", "Geri Yükle": "dashboard.users.restore",
	"Tehlikeli bölge": "common.danger_zone", "Kullanıcıyı Sil": "dashboard.users.delete",
	"Roller": "dashboard.rbac.roles", "Rol Listesi": "dashboard.rbac.role_list", "Yeni Rol": "dashboard.rbac.new_role",
	"Rol Adı": "dashboard.rbac.role_name", "İzinler": "dashboard.rbac.permissions",
	"Tümünü Seç": "common.select_all", "Tümünü Kaldır": "common.clear_all",
	"İzin Listesi": "dashboard.rbac.permission_list", "Yeni İzin": "dashboard.rbac.new_permission",
	"Bildirim Gönder": "dashboard.notifications.send", "Toplu Gönderim": "dashboard.notifications.bulk",
	"Dosyadan Toplu Gönderim": "dashboard.notifications.upload", "Kanal": "dashboard.notifications.channel",
	"Alıcı": "dashboard.notifications.recipient", "Alıcılar (her satırda e-posta)": "dashboard.notifications.email_recipients",
	"Alıcılar (her satırda telefon)": "dashboard.notifications.phone_recipients",
	"Tek alıcı":                      "dashboard.notifications.single_recipient", "Tüm aktif kullanıcılar": "dashboard.notifications.all_users",
	"Varsayılan Dil": "dashboard.notifications.default_language", "Gönderim Sonucu": "dashboard.notifications.result",
	"Toplam": "common.total", "Kabul": "common.accepted", "Geçersiz": "common.invalid",
	"Dosya Yükle": "dashboard.uploads.title", "Yüklenen Dosyalar": "dashboard.uploads.uploaded_files",
	"Yeni Yükleme": "dashboard.uploads.new_upload", "Dosyalar": "dashboard.uploads.files",
	"SMS Ayarları": "dashboard.settings.sms.title", "Ödeme Ayarları": "dashboard.settings.payments.title",
	"Sağlayıcı Listesi": "dashboard.settings.provider_list", "Sağlayıcı": "dashboard.settings.provider",
	"Yapılandırma": "dashboard.settings.configuration", "Hazır": "dashboard.settings.ready",
	"Eksik": "dashboard.settings.missing", "Aktif Yap": "dashboard.settings.activate",
	"Ödeme": "dashboard.payments.checkout", "Ödemeler": "dashboard.payments.title",
	"Yeni Tahsilat": "dashboard.payments.new_charge", "Ödeme Listesi": "dashboard.payments.list",
	"Referans": "dashboard.payments.reference", "Tutar": "dashboard.payments.amount",
	"Taksit": "dashboard.payments.installment", "Kart": "dashboard.payments.card",
	"Ödeme Detayı": "dashboard.payments.detail", "3DS Tamamla": "dashboard.payments.complete_3ds",
	"Denetim Kayıtları": "dashboard.audit.title", "Denetim Detayı": "dashboard.audit.detail",
	"Kayıt Listesi": "dashboard.audit.list", "Aksiyon": "dashboard.audit.action",
	"Kaynak": "dashboard.audit.resource", "Aktör": "dashboard.audit.actor",

	// Flash / notice
	"Mesaj okundu olarak işaretlendi":      "dashboard.notice.contact_marked_read",
	"SMS sağlayıcısı güncellendi":          "dashboard.notice.sms_updated",
	"Ödeme sağlayıcısı güncellendi":        "dashboard.notice.payment_updated",
	"3DS tamamlandı":                       "dashboard.notice.3ds_completed",
	"3DS başlatıldı":                       "dashboard.notice.3ds_started",
	"rol oluşturuldu":                      "dashboard.notice.role_created",
	"rol güncellendi":                      "dashboard.notice.role_updated",
	"rol izinleri güncellendi":             "dashboard.notice.role_perms_updated",
	"rol silindi":                          "dashboard.notice.role_deleted",
	"izin oluşturuldu":                     "dashboard.notice.perm_created",
	"izin güncellendi":                     "dashboard.notice.perm_updated",
	"toplu gönderim kuyruğa alındı":        "dashboard.notice.bulk_queued",
	"bildirim gönderildi":                  "dashboard.notice.notification_sent",
	"dosya yüklendi":                       "dashboard.notice.file_uploaded",
	"ad güncellendi":                       "dashboard.notice.name_updated",
	"e-posta adresi güncellendi":           "dashboard.notice.email_updated",
	"telefon numarası güncellendi":         "dashboard.notice.phone_updated",
	"dil tercihi güncellendi":              "dashboard.notice.locale_updated",
	"şifreniz değiştirildi":                "dashboard.notice.password_changed",
	"iki adımlı doğrulama etkinleştirildi": "dashboard.notice.mfa_enabled",
	"iki adımlı doğrulama kapatıldı":       "dashboard.notice.mfa_disabled",
	"bildirim okundu işaretlendi":          "dashboard.notice.notif_marked_read",
	"tüm bildirimler okundu işaretlendi":   "dashboard.notice.notif_marked_all_read",
	"bildirim silindi":                     "dashboard.notice.notif_deleted",
	"tüm bildirimler silindi":              "dashboard.notice.notif_deleted_all",
	"Tümünü okundu yap":                    "dashboard.account.mark_all_read",
	"Tümünü sil":                           "dashboard.account.delete_all_notifications",
	"Bildirimi sil":                        "dashboard.account.delete_notification",
	"kullanıcı başarıyla kaydedildi":       "dashboard.notice.user_registered",
	"kullanıcı rolü güncellendi":           "dashboard.notice.user_role_updated",
	"kullanıcı aktifleştirildi":            "dashboard.notice.user_activated",
	"kullanıcı silindi":                    "dashboard.notice.user_deleted",
	"kullanıcı geri yüklendi":              "dashboard.notice.user_restored",
}
