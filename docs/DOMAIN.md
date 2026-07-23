# Domain ve mimari

## Katmanlar

| Katman | Yol | Sorumluluk |
|---|---|---|
| Domain | `internal/domain/*` | Entity, value object, domain event, repository **port** |
| Application | `internal/application/*` | Use-case, DTO, orchestration, CQRS komut/sorgu |
| Adapters | `internal/adapters/{http,goui,persistence,shared}` | Fiber handler, GoUI, sqlc repo, ortak adapter yardımcıları |
| Infrastructure | `internal/infrastructure/*` | Config, DB, JWT/Argon2/TOTP, SMTP, SMS, ödeme gateway, outbox, cache |
| Bootstrap | `internal/bootstrap` (`Build` + `wire_*.go`) | Manuel DI; composition root parçalı wiring |

Domain, application dışına bağımlı olmaz. Application yalnızca domain portlarına ve kendi DTO’larına bakar.

## Bounded context’ler

| Context | Domain | Application | Not |
|---|---|---|---|
| User | `domain/user` | `application/user` | CRUD, profil, soft-delete |
| Auth | `domain/auth` | `application/auth` | Token, MFA, OAuth olayları |
| Authz / RBAC | `domain/rbac` | `application/authz` | Rol/izin aggregate; çözümleme authz’de |
| Notification | `domain/notification` | `application/notification` | Dispatcher, inbox, bulk |
| Settings | `domain/settings` | `application/settings` | Aktif SMS/ödeme sağlayıcı |
| Payment | `domain/payment` | `application/payment` | 3DS, transaction |
| Contact | `domain/contact` | `application/contact` | Public form + admin kutu |
| Upload | `domain/upload` | `application/upload` | Güvenli dosya yükleme |
| Audit | — | `application/audit` | Domain aggregate yok; uygulama servisi |
| Idempotency / Outbox | — | `application/{idempotency,outbox}` | Altyapı destekli |

Paylaşılan domain tipleri: `domain/shared`.

## `pkg/` sözleşmeleri

Bunlar **domain değildir**; çapraz yardımcı paketlerdir.

| Paket | Kullanım |
|---|---|
| `pkg/rbac` | `Perm*` sabitleri, `Catalog`, `Checker` — aggregate `domain/rbac`, use-case `application/authz` |
| `pkg/recipients` | CSV/XLSX **alıcı içe aktarma** (bildirim bulk) — iletişim formu `domain/contact` |
| `pkg/tabular` | Liste **dışa aktarma** (CSV/XLSX) |
| `pkg/i18n` | Locale dosyaları (`locales/tr.json`, `en.json`) |
| `pkg/validation` | go-playground tag kayıtları + alan mesajları |
| `pkg/pagination` | Offset/limit yardımcıları |
| `pkg/fieldenc` | Hassas alan şifreleme (ödeme kartı vb.) |

`pkg/user` yoktur; kullanıcı yalnızca `internal/domain/user` altındadır.

## Yüzeyler (adapters)

### HTTP API — `internal/adapters/http`

- Kayıt: `server.go` → `/api/v1/...`
- Handler’lar: `handler/*.go`
- Middleware: auth, i18n, idempotency, permission

Özet gruplar:

- Public: health, OpenAPI, contact submit, auth login/refresh/oauth, payment callback/webhook
- Protected: `/me`, users, notifications send/bulk, rbac, settings, payments, audit, contacts, uploads

### GoUI panel — `internal/adapters/goui`

- Sayfa rotaları: `routes.go` → `pageRoutes()`
- Controller’lar: `controller_*.go`
- Export: `export_routes.go`
- Yardımcı (OAuth, dil, 3DS start): `utility_routes.go`

Web oturumu HttpOnly cookie; API Bearer JWT. İkisi aynı application use-case’lerini çağırır.

Örnek panel yolları: `/dashboard`, `/dashboard/users`, `/dashboard/notifications/bulk`, `/dashboard/payments/checkout`, `/dashboard/audit/logs`.

## Veri

| Parça | Yol |
|---|---|
| Migration | `migrations/000001_schema.*`, `000002_seed.*` (embed: `migrations.go`) |
| sqlc kaynak | `db/queries/*.sql` |
| sqlc çıktı | `internal/adapters/persistence/postgres/db/` |

Geliştirmede şema tek dosyada tutulur (`000001`). Canlıya çıktıktan sonra incremental migration eklenir.

## Bildirim akışı (kısa)

1. Use-case → `Dispatcher` (kanal: `inapp` | `sms` | `email`)
2. Alıcı çözümü: kullanıcı ID, e-posta, telefon veya `pkg/recipients` parse
3. In-app: DB kaydı + `/api/v1/ws` üzerinden `inbox.updated` (panel/mobil/API)
4. SMS/e-posta: aktif sağlayıcı (settings) + worker/async

## İlişkili dokümanlar

- İlk okuma / golden path → [READING.md](READING.md)
- Context haritaları → [map/](map/) (`user`, `auth`, `contact`, `notification`, `audit`)
- Auth/RBAC ayrıntı → [AUTH.md](AUTH.md)
- Ödeme → [PAYMENT.md](PAYMENT.md)
- Yeni modül → [YENI_MODUL_EKLEME.md](YENI_MODUL_EKLEME.md)
