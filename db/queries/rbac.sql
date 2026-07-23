-- Dinamik RBAC. İzinler DB'de; roller runtime'da yönetilir.

-- name: CreatePermission :exec
INSERT INTO permissions (id, name, description)
VALUES ($1, $2, $3);

-- name: PermissionExists :one
SELECT EXISTS(SELECT 1 FROM permissions WHERE name = $1) AS exists;

-- name: UpdatePermissionDescription :execrows
UPDATE permissions SET description = $2 WHERE name = $1;

-- name: ListPermissions :many
SELECT name, description FROM permissions ORDER BY name;

-- name: ListRoles :many
SELECT name, description, is_system FROM roles ORDER BY name;

-- name: GetRoleByName :one
SELECT name, description, is_system FROM roles WHERE name = $1;

-- name: RoleExists :one
SELECT EXISTS(SELECT 1 FROM roles WHERE name = $1) AS exists;

-- name: CreateRole :exec
INSERT INTO roles (id, name, description, is_system)
VALUES ($1, $2, $3, $4);

-- name: UpdateRoleDescription :execrows
UPDATE roles SET description = $2, updated_at = now() WHERE name = $1;

-- name: DeleteRole :execrows
-- Sistem rolleri (admin/user) silinemez.
DELETE FROM roles WHERE name = $1 AND is_system = FALSE;

-- name: ListPermissionsForRole :many
SELECT p.name
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN roles r ON r.id = rp.role_id
WHERE r.name = $1
ORDER BY p.name;

-- name: ClearRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = $1);

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = $1 AND p.name = $2
ON CONFLICT DO NOTHING;

-- name: GrantAllPermissionsToRole :exec
-- GrantAllPermissionsToRole, verilen role DB'deki tüm izinleri atar (admin senkronu).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = $1
ON CONFLICT DO NOTHING;

-- name: CountUsersWithRole :one
SELECT COUNT(*)::bigint AS count
FROM users
WHERE role = $1 AND deleted_at IS NULL;
