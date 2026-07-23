# Map: User

Kullanıcı bounded context — dosyaları tek bakışta.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/user/` |
| Application | `internal/application/user/` |
| HTTP | `internal/adapters/http/handler/user_handler.go` |
| GoUI | `internal/adapters/goui/controller_account_users.go` |
| Persistence | `internal/adapters/persistence/postgres/user_repository.go` |
| SQL | `db/queries/` (users) |
| Wiring | `bootstrap/wire_user.go`, `wire_http.go`, `wire_goui.go` (`UserDeps`) |

## Application (use-case’ler)

| Dosya | Tipik iş |
|---|---|
| `service.go` | `Service` facade — transport'ın okuma yüzeyi |
| `query.go` | `GetHandler`, `ListHandler` |
| `command.go` | Register, Activate, Delete, Restore, Change* |
| `access.go` | Liste / profil / rol yetki kontrolleri |
| `dto.go` | Dışarıya giden view/DTO |
| `locale_policy.go` | Tercih dili |
| `ports.go` | Application’ın dış portları (gerekirse) |

## HTTP yüzeyleri

Kayıt: `internal/adapters/http/server.go` → `/api/v1/users/...`

| Method | Path | Handler metodu | İzin (özet) |
|---|---|---|---|
| POST | `/users/` | Register | public |
| GET | `/users/` | List | `users:list` |
| GET | `/users/:id` | Get | auth |
| GET | `/users/me` | Me | auth |
| PATCH | `/users/me/*` | ChangeMy* / ChangeLocale | auth |
| POST | `/users/create` | AdminCreate | `users:list` |
| POST | `/users/:id/activate` | Activate | `users:activate` |
| DELETE | `/users/:id` | Delete | `users:delete` |
| POST | `/users/:id/restore` | Restore | `users:restore` |
| PATCH | `/users/:id/role` | ChangeRole | `users:role:change` |

## Panel yüzeyleri

| Path | Screen | Controller (dosya içi) |
|---|---|---|
| `/dashboard/users` | `users` | list |
| `/dashboard/users/new` | `user-new` | new |
| `/dashboard/users/:id` | `user-show` | show |
| `/dashboard/account` | `account` | hesap (aynı dosya ailesi) |

Rota tablosu: `adapters/goui/routes.go` → `pageRoutes()`.

## Uçtan uca (liste)

API: [READING.md — Golden path A](../READING.md#golden-path-a--kullanıcı-listesi-api)  
Panel: [READING.md — Golden path B](../READING.md#golden-path-b--aynı-liste-panel)

## İlişkili

- Auth oturumu / JWT → [map/auth.md](auth.md)
- RBAC izin sabitleri → `pkg/rbac`
- Mimari → [DOMAIN.md](../DOMAIN.md)
