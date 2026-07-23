# Map: Auth

Kimlik doğrulama bounded context — dosyaları tek bakışta.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/auth/` |
| Application | `internal/application/auth/` — **önce** `facade.go` (`Service`) |
| HTTP | `handler/auth_handler.go` (+ account / mfa / oauth) |
| GoUI | `controller_public_auth.go`, account MFA |
| Infra | `infrastructure/security`, `infrastructure/oauth` |
| Wiring | `wire_auth.go` → `authService` (API) + `authServiceWeb` (panel OAuth) |

## Application

| Dosya | İş |
|---|---|
| `facade.go` | Transport facade: Login, Session, MFA, OAuth, şifre, e-posta |
| `service.go` | `LoginHandler` (CQRS) |
| `session.go` | Token oturumu |
| `password.go` / `verify.go` / `mfa.go` / `oauth.go` | Diğer use-case’ler |

## Uçtan uca (login)

1. Transport → `Service.Login`
2. `LoginHandler.Handle`
3. `SessionManager` token üretir  

Detay: [AUTH.md](../AUTH.md), [READING.md](../READING.md#golden-path-c--giriş-api--panel).
