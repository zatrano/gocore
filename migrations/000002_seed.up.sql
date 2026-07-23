-- Başlangıç verileri: sistem rolleri, tüm izinler (audit:list dahil) ve varsayılan ayarlar.

INSERT INTO roles (id, name, description, is_system)
VALUES
    (gen_random_uuid(), 'admin', 'Tüm yetkilere sahip sistem yöneticisi', TRUE),
    (gen_random_uuid(), 'user', 'Standart kullanıcı', TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (id, name, description) VALUES
    (gen_random_uuid(), 'users:list', 'Kullanıcıları listeleme'),
    (gen_random_uuid(), 'users:read', 'Başka kullanıcıların profilini görüntüleme'),
    (gen_random_uuid(), 'users:activate', 'Kullanıcı hesabı aktifleştirme'),
    (gen_random_uuid(), 'users:delete', 'Kullanıcı silme (soft delete)'),
    (gen_random_uuid(), 'users:restore', 'Silinmiş kullanıcıyı geri getirme'),
    (gen_random_uuid(), 'users:role:change', 'Kullanıcı rolü değiştirme'),
    (gen_random_uuid(), 'users:email:change:any', 'Herhangi bir kullanıcının e-postasını değiştirme'),
    (gen_random_uuid(), 'uploads:create', 'Dosya yükleme'),
    (gen_random_uuid(), 'rbac:manage', 'Rol ve izinleri yönetme'),
    (gen_random_uuid(), 'notifications:send', 'Elle ve toplu bildirim/SMS/e-posta gönderme'),
    (gen_random_uuid(), 'notifications:settings', 'SMS sağlayıcı ve bildirim ayarlarını yönetme'),
    (gen_random_uuid(), 'payments:charge', '3DS tahsilat (BIN, başlatma, tamamlama)'),
    (gen_random_uuid(), 'payments:list', 'Ödeme listesi ve detay görüntüleme'),
    (gen_random_uuid(), 'audit:list', 'Denetim kayıtlarını listeleme'),
    (gen_random_uuid(), 'contacts:list', 'İletişim mesajlarını listeleme')
ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO app_settings (key, value)
VALUES
    ('sms.active_provider', 'netgsm'),
    ('payment.active_provider', 'iyzico')
ON CONFLICT (key) DO NOTHING;
