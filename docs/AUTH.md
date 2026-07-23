# Kimlik doğrulama ve yetkilendirme

## Yüzeyler

| Yüzey | Oturum | Önek |
|---|---|---|
| GoUI | HttpOnly cookie (`access` / `refresh`) | `/auth/*`, `/dashboard/*`, `/goui/ws` |
| API | `Authorization: Bearer <access>` | `/api/v1/auth/*`, korumalı `/api/v1/*` |
| Canlı olay | Bearer, `?access_token=` veya cookie | `/api/v1/ws` |

Aynı `application/auth` ve `application/user` use-case’leri kullanılır.

Panel bildirim zili ve mobil istemciler **aynı** `/api/v1/ws` hub’ından `inbox.updated` alır; GoUI WS yalnızca UI patch/toast içindir.

## JWT

| Token | Amaç | Yapılandırma |
|---|---|---|
| Access | Kısa ömürlü API/panel erişimi | `AUTH_ACCESS_TOKEN_TTL` |
| Refresh | Rotation ile yenileme | `AUTH_REFRESH_TOKEN_TTL` |

- Issuer / audience: `AUTH_JWT_ISSUER`, `AUTH_JWT_AUDIENCE`
- Secret: `AUTH_JWT_SECRET` (production ≥ 32 karakter)
- Refresh reuse tespiti → ilgili oturum ailesi iptal
- Logout → refresh iptali

## API uçları (`/api/v1/auth`)

| Metot | Yol | Not |
|---|---|---|
| POST | `/login` | E-posta + parola (+ Turnstile) |
| POST | `/refresh` | Yeni access/refresh |
| POST | `/logout` | |
| POST | `/forgot-password` | |
| POST | `/reset-password` | |
| POST | `/verify-email` | |
| POST | `/resend-verification` | |
| GET | `/oauth/:provider` | `google` \| `github` |
| GET | `/oauth/:provider/callback` | |
| POST | `/change-password` | Auth gerekli |
| POST | `/mfa/setup` \| `/mfa/enable` \| `/mfa/disable` | Auth gerekli |
| GET | `/permissions` | Oturum sahibinin izinleri |

Web karşılıkları: `/auth/login`, `/auth/register`, `/auth/forgot-password`, … ve OAuth callback’ler `utility_routes` üzerinden.

## Brute-force

`LoginGuard`: e-posta ve IP bazlı deneme sınırı.

- `AUTH_MAX_LOGIN_ATTEMPTS`
- `AUTH_LOCKOUT_DURATION`
- Altyapı: `infrastructure/security` (IP rate limiter)

## MFA (TOTP)

RFC 6238. Secret alan şifrelemesi: `AUTH_MFA_ENCRYPTION_KEY` (`openssl rand -base64 32`).

Akış: setup → kullanıcı authenticator’a ekler → enable (kod doğrulama) → login’de ikinci adım.

## E-posta token’ları

Doğrulama / şifre sıfırlama: tek kullanımlık, SHA-256 hash’li, süreli.

- `AUTH_VERIFY_TOKEN_TTL`, `AUTH_RESET_TOKEN_TTL`
- Link kökü: `AUTH_EMAIL_LINK_BASE_URL`
- SMTP yoksa `LogMailer` (yalnızca log)

## OAuth

Env boşsa sağlayıcı kapalıdır.

| Değişken | Açıklama |
|---|---|
| `OAUTH_CALLBACK_BASE_URL` | API callback kökü |
| `OAUTH_GOOGLE_*` / `OAUTH_GITHUB_*` | Client id/secret |

Web callback örnek: `http://localhost:8080/auth/oauth/google/callback`  
API callback örnek: `http://localhost:8080/api/v1/auth/oauth/google/callback`

Davranış: bul-veya-oluştur (`user` rolü).

## Turnstile

Public formlar (giriş, kayıt, iletişim vb.): `SEC_TURNSTILE_SITE_KEY` / `SEC_TURNSTILE_SECRET_KEY`.  
Development’ta boş → doğrulama kapalı. Production’da zorunlu.

## RBAC

### İzin kataloğu (`pkg/rbac`)

| İzin | Anlam |
|---|---|
| `users:list` | Kullanıcı listesi |
| `users:read` | Başka kullanıcı profili |
| `users:activate` | Hesap aktifleştirme |
| `users:delete` / `users:restore` | Soft-delete / geri alma |
| `users:role:change` | Rol atama |
| `users:email:change:any` | Başkasının e-postası |
| `uploads:create` | Dosya yükleme |
| `rbac:manage` | Rol/izin yönetimi |
| `notifications:send` | Tekil/toplu bildirim |
| `notifications:settings` | SMS/ödeme ayar seçimi |
| `payments:charge` | 3DS tahsilat |
| `payments:list` | İşlem listesi |
| `audit:list` | Denetim kayıtları |
| `contacts:list` | İletişim mesajları |

Sistem rolleri (seed): `admin` (tümü), `user` (sahiplik odaklı).

### Çalışma şekli

1. Açılışta katalog DB ile senkron (`Catalog`)
2. `application/authz` izin çözer (TTL cache; mutasyonda invalidate)
3. Middleware: `RequirePermission` (HTTP + GoUI route `permission` alanı)

Rol CRUD: `/api/v1/rbac/...` ve panel `/dashboard/rbac/*`.

## İlgili kod

- `internal/application/auth`
- `internal/application/authz`
- `internal/adapters/http/handler/auth_*.go`
- `internal/adapters/http/middleware/auth.go`
- `internal/adapters/goui/controller_public_auth.go`
- `pkg/rbac`
