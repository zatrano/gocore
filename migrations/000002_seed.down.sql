DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE name = 'admin');

DELETE FROM permissions
WHERE name IN (
    'users:list', 'users:read', 'users:activate', 'users:delete', 'users:restore',
    'users:role:change', 'users:email:change:any',
    'uploads:create', 'rbac:manage', 'notifications:send', 'notifications:settings',
    'payments:charge', 'payments:list', 'audit:list', 'contacts:list'
);

DELETE FROM roles WHERE name IN ('admin', 'user');

DELETE FROM app_settings
WHERE key IN ('sms.active_provider', 'payment.active_provider');
