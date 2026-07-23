# Map: RBAC

Dinamik rol / izin yönetimi bounded context.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/rbac/` |
| Application | `internal/application/authz/` — **önce** `service.go` |
| HTTP | `internal/adapters/http/handler/rbac_handler.go` |
| GoUI | `internal/adapters/goui/controller_rbac_notifications.go` |
| Persistence | `adapters/persistence/postgres/role_repository.go` |
| Wiring | `wire_authz.go`; `wire_http.go`, `wire_goui.go` |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: izin/rol CRUD, rol-izin atama |
| `resolver.go` | `Resolver` — oturum izin kontrolü (bellek içi TTL önbellek) |
| `dto.go` | `PermissionInfo`, `RoleInfo` |

`Resolver` tek süreç dağıtımında process belleğinde önbellek tutar; çoklu instance için paylaşımlı invalidation gerekir (şimdilik yok).

## HTTP yüzeyleri

Kayıt: `server.go` → `/api/v1/rbac/...` (izin: `rbac:manage`)

| Method | Path | Handler |
|---|---|---|
| GET/POST | `/rbac/permissions` | List / Create permission |
| PATCH | `/rbac/permissions/:name` | Update permission |
| GET/POST | `/rbac/roles` | List / Create role |
| GET/PATCH/DELETE | `/rbac/roles/:name` | Get / Update / Delete role |
| PUT | `/rbac/roles/:name/permissions` | SetPermissions |

## Panel yüzeyleri

| Path | Screen |
|---|---|
| `/dashboard/rbac/roles` | `roles` |
| `/dashboard/rbac/roles/new` | `role-new` |
| `/dashboard/rbac/roles/:name` | `role-show` |
| `/dashboard/rbac/permissions` | `permissions` |

## İlişkili

- Oturum / JWT → [map/auth.md](auth.md)
- İzin sabitleri → `pkg/rbac`
- Derinlik → [AUTH.md](../AUTH.md)
