# GoCore

Clean Architecture + Hexagonal + DDD + CQRS ile yazılmış kurumsal Go platformu.
Tek süreçte **JSON REST API** (`/api/v1`) ve **GoUI** admin paneli çalışır.

| | |
|---|---|
| Modül | `github.com/zatrano/gocore` |
| Go | 1.25.12 |
| HTTP | Fiber v3 |
| UI | GoUI v1 (server-driven + WebSocket; public sayfalar ModeSEO) |
| DB | PostgreSQL · pgxpool · sqlc |

## Özellikler

- Kimlik: JWT (access/refresh + rotation), MFA TOTP, e-posta doğrulama, şifre sıfırlama, OAuth (Google/GitHub), Turnstile
- Kullanıcı yönetimi: liste, aktivasyon, soft-delete, rol değişimi
- Dinamik RBAC: roller/izinler DB’de; `RequirePermission` ile koruma
- Bildirimler: in-app / SMS / e-posta; tekil + toplu (JSON, CSV, Excel)
- Ödeme: Iyzico / Moka 3DS, BIN, webhook, işlem listesi
- İletişim formu, ayarlar (aktif SMS/ödeme sağlayıcı), audit, güvenli upload, i18n (`tr`/`en`)

## Hızlı başlangıç

```bash
cp .env.example .env
# PostgreSQL ayarlarını düzenle

go run ./cmd/migrate up
go run ./cmd/seed
go run ./cmd/server
```

| Adres | Açıklama |
|---|---|
| http://localhost:8080/ | Web / panel |
| http://localhost:8080/api/v1 | REST API |
| http://localhost:8080/docs | Swagger UI |
| http://localhost:8080/openapi.yaml | OpenAPI 3.1 |

Seed admin (geliştirme): `zatrano@zatrano.com` / `ZATRANO`

## Proje yapısı

```
cmd/                  server, migrate, seed
internal/
  domain/             entity, VO, repository portları
  application/        use-case / CQRS
  adapters/
    http/             JSON API (Fiber)
    goui/             panel sayfaları
    persistence/      sqlc + postgres repo
  infrastructure/     config, güvenlik, e-posta, SMS, ödeme, outbox…
  bootstrap/          composition root (Build + wire_*.go)
pkg/                  rbac, recipients, tabular, i18n, validation…
migrations/           000001_schema + 000002_seed
db/queries/           sqlc kaynak SQL
api/openapi.yaml      API sözleşmesi
```

Bağımlılık yönü: **domain ← application ← adapters**; infrastructure dışarıda; hepsi `bootstrap` içinde bağlanır.

**İlk okuma:** [docs/READING.md](docs/READING.md) — klasör haritası, isim sözlüğü, golden path (user list + login).

## Geliştirme

```bash
go test ./...
go test -race ./...
golangci-lint run
sqlc generate
```

CI (`.github/workflows/ci.yml`): lint, race test, govulncheck, gosec, build.

## Dokümantasyon

| Dosya | İçerik |
|---|---|
| [docs/READING.md](docs/READING.md) | **İlk okuma** — harita, sözlük, golden path |
| [docs/map/user.md](docs/map/user.md) | User context dosya haritası |
| [docs/map/auth.md](docs/map/auth.md) | Auth context dosya haritası |
| [docs/map/contact.md](docs/map/contact.md) | Contact haritası |
| [docs/map/notification.md](docs/map/notification.md) | Notification haritası |
| [docs/map/audit.md](docs/map/audit.md) | Audit haritası |
| [docs/map/payment.md](docs/map/payment.md) | Payment (3DS) haritası |
| [docs/map/settings.md](docs/map/settings.md) | Settings haritası |
| [docs/map/upload.md](docs/map/upload.md) | Upload haritası |
| [docs/map/rbac.md](docs/map/rbac.md) | RBAC haritası |
| [docs/DOMAIN.md](docs/DOMAIN.md) | Mimari, bounded context’ler, paket sözleşmeleri |
| [docs/AUTH.md](docs/AUTH.md) | Auth, oturum, MFA, OAuth, RBAC izinleri |
| [docs/PAYMENT.md](docs/PAYMENT.md) | 3DS, sağlayıcılar, webhook |
| [docs/VALIDATION.md](docs/VALIDATION.md) | Girdi doğrulama ve i18n hataları |
| [docs/DEPLOY.md](docs/DEPLOY.md) | Ortam, migration, production checklist |
| [docs/YENI_MODUL_EKLEME.md](docs/YENI_MODUL_EKLEME.md) | Yeni bounded context ekleme adımları |

Yapılandırma anahtarları: [`.env.example`](.env.example) (`APP_*`, `HTTP_*`, `DB_*`, `AUTH_*`, `OAUTH_*`, `SEC_*`, `I18N_*`, `SMTP_*`, `CONTACT_*`, `NOTIFY_*`, `PAYMENT_*`).
