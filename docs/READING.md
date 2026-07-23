# GoCore’u okuma rehberi

Bu dosya, mimariyi bozmadan projeyi **insan olarak** takip etmeyi kolaylaştırır.
Derin mimari → [DOMAIN.md](DOMAIN.md). Yeni modül yazma → [YENI_MODUL_EKLEME.md](YENI_MODUL_EKLEME.md).

## 5 dakikada klasör haritası

```
cmd/server          → giriş; bootstrap.Run
internal/bootstrap  → composition root (Build + wire_*.go)
internal/domain/X   → iş kuralları, repository PORT (arayüz)
internal/application/X → use-case (CQRS: komut / sorgu)
internal/adapters/http     → JSON API (Fiber handler)
internal/adapters/goui     → panel sayfaları (Controller)
internal/adapters/persistence/postgres → sqlc repo (port implementasyonu)
internal/infrastructure    → JWT, mail, SMS, ödeme, cache…
pkg/                → çapraz yardımcı (domain değil)
db/queries/         → sqlc SQL kaynağı
migrations/         → şema
api/openapi.yaml    → API sözleşmesi
```

**Bağımlılık yönü (ezber):**  
`domain ← application ← adapters` · altyapı dışarıda · hepsi `bootstrap`’ta bağlanır.

Context bazlı dosya listeleri: [map/user.md](map/user.md), [map/auth.md](map/auth.md), [map/contact.md](map/contact.md), [map/notification.md](map/notification.md), [map/audit.md](map/audit.md), [map/payment.md](map/payment.md), [map/settings.md](map/settings.md), [map/upload.md](map/upload.md), [map/rbac.md](map/rbac.md).

---

## Application Service yüzeyi (standart)

Transport (HTTP / GoUI) **önce** `application/<context>.Service` (veya mevcut `*Service`) okur.
CQRS `*Handler` dosyaları içeride kalır; iş kuralı orada.

| Context | Facade dosyası | GoUI / HTTP alanı |
|---|---|---|
| User | `application/user/service.go` | `Users` |
| Auth | `application/auth/facade.go` | `Auth` |
| Contact | `application/contact/service.go` | `Contacts` |
| Notification | `application/notification/service.go` | `Notifications` |
| Audit | `application/audit/service.go` | `Audit` |
| Settings | `application/settings/service.go` | `Settings` |
| Payment | `application/payment` (`ThreeDSService`) | `ThreeDSSvc` |
| Upload | `application/upload/service.go` | `Upload` |
| Authz / RBAC | `application/authz/service.go` | `Authz` |

Yeni context eklerken: CQRS handler’ları yaz → `NewService` facade → transport yalnızca Service alsın.

---

## İsim sözlüğü (tek dil)

| İsim | Nerede | Ne işe yarar | Ne değildir |
|---|---|---|---|
| **Use-case / `*Handler` (application)** | `internal/application/...` | Bir iş kuralı adımı (Login, ListUsers…) | HTTP veya HTML bilmez |
| **HTTP handler** | `internal/adapters/http/handler` | JSON isteği parse eder → use-case çağırır | Domain yazmaz |
| **GoUI Controller** | `internal/adapters/goui` | Sayfa state: Mount / Render / HandleEvent | REST handler değildir |
| **Repository port** | `internal/domain/...` | Kalıcılık arayüzü | SQL / GORM yok |
| **Repository impl** | `adapters/persistence/postgres` | sqlc ile portu doldurur | İş kuralı koyma |
| **Deps** | handler / GoUI / use-case | Constructor injection grupları | God-container değil |
| **Service (facade)** | `application/<ctx>/service.go` veya `facade.go` | Transport’un baktığı tek yüzey; içeride CQRS | İş kuralını kopyalamaz, delege eder |
| **wire_*.go** | `internal/bootstrap` | Composition root parçaları | İş mantığı yok |

> Application katmanında tip adı hâlâ `*Handler` olabilir (CQRS kalıbı). Okurken bunları **use-case** diye düşün; HTTP handler ile karıştırma.

---

## Bir istek nasıl akar? (genel)

```
İstek
  → Fiber middleware (auth, i18n, RBAC, idempotency…)
  → HTTP handler  VEYA  GoUI Controller
  → application.<Context>Service  (facade)
  → CQRS use-case (command/query handler)
  → domain port (Repository, …)
  → postgres/sqlc adapter
  → yanıt: JSON  |  HTML/WS diff
```

API ve panel **aynı Service / use-case’i** çağırır; iş kuralı tek yerde kalır.

---

## Golden path A — Kullanıcı listesi (API)

`GET /api/v1/users` (izin: `users:list`)

Okuma sırası:

1. Rota — [`internal/adapters/http/server.go`](../internal/adapters/http/server.go) (`protected.Get("/", … List)`)
2. Transport — [`internal/adapters/http/handler/user_handler.go`](../internal/adapters/http/handler/user_handler.go) → `List`
3. Use-case — [`internal/application/user/service.go`](../internal/application/user/service.go) (`Service` facade) → [`query.go`](../internal/application/user/query.go) (`ListHandler`)
4. Erişim — [`internal/application/user/access.go`](../internal/application/user/access.go)
5. Port — [`internal/domain/user`](../internal/domain/user) (`Repository`)
6. Impl — [`internal/adapters/persistence/postgres/user_repository.go`](../internal/adapters/persistence/postgres/user_repository.go)
7. SQL — [`db/queries/`](../db/queries/) (users ile ilgili sorgular)
8. Wiring — [`internal/bootstrap/wire_user.go`](../internal/bootstrap/wire_user.go) + [`wire_http.go`](../internal/bootstrap/wire_http.go)

Detaylı dosya haritası: [map/user.md](map/user.md).

**Sayfalama:** Panel `page` + `limit` (offset) kullanır. REST API aynı uç noktada isteğe bağlı `cursor` query parametresi ile keyset sayfalama yapabilir; yanıtta `next_cursor` doluysa sonraki sayfa için geçirilir. Offset alanları geriye dönük uyumluluk için korunur.

---

## Golden path B — Aynı liste (panel)

`GET /dashboard/users` → screen `users`

Okuma sırası:

1. Rota — [`internal/adapters/goui/routes.go`](../internal/adapters/goui/routes.go) (`pageRoutes`, path `/dashboard/users`)
2. Factory — [`internal/adapters/goui/controllers.go`](../internal/adapters/goui/controllers.go) → `controllerFor`
3. Sayfa — [`internal/adapters/goui/controller_account_users.go`](../internal/adapters/goui/controller_account_users.go) (`usersListController`)
4. Use-case — yine `application/user` `Service` → `List` (API ile aynı)
5. Wiring — [`internal/bootstrap/wire_goui.go`](../internal/bootstrap/wire_goui.go) (`UserDeps`)

---

## Golden path C — Giriş (API + panel)

| Yüzey | Giriş noktası |
|---|---|
| API | `POST /api/v1/auth/login` → `handler/auth_handler.go` → `application/auth` `Service.Login` → `LoginHandler` |
| Panel | `/auth/login` → `controller_public_auth.go` → aynı `Service.Login` (web OAuth’lu `authServiceWeb`) |

Dosya haritası: [map/auth.md](map/auth.md). Derinlik: [AUTH.md](AUTH.md).

---

## Tek süreç bellek (Redis yok)

Tek instance dağıtımda **Redis gerekmez**. Aşağıdaki bileşenler process içi bellek kullanır:

| Bileşen | Konum | Not |
|---|---|---|
| JWT refresh token store | `security.MemoryTokenStore` | Oturum rotation |
| Login guard / IP limiter | `security.MemoryLoginGuard`, `MemoryIPRateLimiter` | Brute-force |
| Authz `Resolver` | `application/authz/resolver.go` | Rol-izin TTL önbellek |
| Settings `Service` | `application/settings/service.go` | Platform ayarları önbellek |

Yatay ölçek (birden fazla süreç) planlandığında paylaşımlı store + invalidation gerekir — şu an kapsam dışı. Ayrıntı: [DEPLOY.md](DEPLOY.md).

---

## Smoke / golden-path testler

Hafif doğrulamalar (tam DB bootstrap şart değil):

| Test | Paket | Ne doğrular |
|---|---|---|
| Cursor encode/decode | `pkg/pagination/cursor_test.go` | Keyset imleci roundtrip |
| Keyset SQL üretimi | `adapters/persistence/postgres/user_keyset_test.go` | sqlc keyset sorguları derlenir |
| User list yüzeyi | `application/user/smoke_test.go` | `ListQuery.Cursor`, `Service.List` |

Tam entegrasyon için `bootstrap/testkit` ile fake port kullanın; production wiring `wire_*.go` dosyalarında kalır.

```bash
go test ./pkg/pagination/... ./internal/application/user/... ./internal/adapters/persistence/postgres/... -count=1
```

---

## “Nereye bakayım?” hızlı tablo

| Soru | İlk dosya |
|---|---|
| Bu URL hangi koda gidiyor? (API) | `adapters/http/server.go` |
| Bu URL hangi ekran? (panel) | `adapters/goui/routes.go` |
| İş kuralı nerede? | `application/<context>/service.go` (facade) → CQRS dosyaları |
| Entity / invariant? | `domain/<context>/` |
| SQL? | `db/queries/` + `persistence/postgres/` |
| Bağımlılık nasıl bağlandı? | `bootstrap/wire_*.go` |
| İzin sabiti? | `pkg/rbac` |
| Yeni context nasıl eklenir? | [YENI_MODUL_EKLEME.md](YENI_MODUL_EKLEME.md) |

---

## Yazma checklist (kısa)

Yeni özellik veya düzeltme:

1. Context’i bul (`user`, `auth`, `payment`…) — yoksa [YENI_MODUL_EKLEME.md](YENI_MODUL_EKLEME.md)
2. Domain gerekirse entity/port
3. Application use-case (komut veya sorgu)
4. Persistence (sqlc + repo) gerekirse
5. HTTP ve/veya GoUI transport
6. RBAC izni gerekirse `pkg/rbac` + route
7. `bootstrap` ilgili `wire_*.go`
8. OpenAPI / i18n gerekirse güncelle
9. `go test` ilgili paketler

Testte full bootstrap şart değil — [bootstrap/testkit](../internal/bootstrap/testkit/doc.go).
