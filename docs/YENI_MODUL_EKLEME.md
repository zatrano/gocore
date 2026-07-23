# Yeni modül (bounded context) ekleme

Bu rehber, GoCore’a yeni bir domain alanı eklerken izlenecek sırayı verir.
Örnek isim: **Note** (`notes`).

## Okuma sırası (yazmadan önce)

1. [READING.md](READING.md) — harita, isim sözlüğü, golden path  
2. Mevcut benzer context — [map/user.md](map/user.md) veya [map/auth.md](map/auth.md)  
3. [DOMAIN.md](DOMAIN.md) — katman kuralları  

Yeni context ekledikten sonra `docs/map/<context>.md` kısa haritasını da ekle (isteğe bağlı ama önerilir).

## Yazma checklist

- [ ] Domain + port (`internal/domain/...`)
- [ ] Migration + `db/queries` + `sqlc generate`
- [ ] Persistence repo (`adapters/persistence/postgres`)
- [ ] Application use-case’ler (`application/...`) + **Service facade**
- [ ] RBAC izni (`pkg/rbac` + route) — gerekiyorsa
- [ ] HTTP handler + `server.go` grubu (handler’a yalnızca Service ver)
- [ ] GoUI route + controller — gerekiyorsa (Deps’e Service)
- [ ] Bootstrap ilgili `wire_*.go` (+ GoUI `*Deps` grubu)
- [ ] OpenAPI / i18n — gerekiyorsa
- [ ] Test + `docs/map/` veya DOMAIN tablosu

## 1. Domain

`internal/domain/note/`

- `entity.go` — aggregate / entity, invariant’lar
- `repository.go` — port arayüzü (infrastructure bilmez)
- Gerekirse `events.go`, value object’ler

Domain yalnızca kendi paketine ve `domain/shared`’e bağımlı olsun.

## 2. Migration + sqlc

1. `migrations/` altına incremental SQL (`up` / `down`)
2. `db/queries/notes.sql` — sqlc sorguları
3. `sqlc generate` → çıktı `internal/adapters/persistence/postgres/db/`

Geliştirmede henüz canlıya çıkmadıysanız şemayı `000001_schema` içinde birleştirmeyi tercih edebilirsiniz; canlı sonrası yalnızca incremental.

## 3. Persistence adapter

`internal/adapters/persistence/postgres/note_repository.go`

- sqlc `Queries` → domain entity map
- `domain/note.Repository` implementasyonu

## 4. Application

`internal/application/note/`

- Komut / sorgu use-case’leri (CQRS `*Handler`)
- **`service.go` facade** — HTTP/GoUI yalnızca buna bağlanır
- DTO’lar (HTTP/GoUI’ye sızmayan domain tipleri)
- Port’lara bağımlılık (repo, clock, event publisher…)

Gerekirse `application/shared` servis kayıtlarına ekleyin.

## 5. RBAC (korumalıysa)

1. `pkg/rbac` — `PermNotesList` vb. sabit + `Catalog` girdisi
2. Seed / sync ile DB’ye yazıldığını doğrula
3. Route’larda `RequirePermission` / GoUI `permission` alanı

## 6. HTTP handler

`internal/adapters/http/handler/note_handler.go`  
`server.go` içinde `/api/v1/notes` grubunu kaydet.

- Public / protected ayrımı
- Mutasyonlarda idempotency middleware (projede kullanılan kalıp)
- Problem JSON + validation hataları

OpenAPI: `api/openapi.yaml` güncelle.

## 7. GoUI (dashboard)

1. `routes.go` → `pageRoutes()` satırı (`screen`, `path`, `permission`, …)
2. Factory: `controllers.go` zincirindeki ilgili `*Controller(screen)` switch’ine tip ekle
3. Controller (`controller_*.go`): `Mount` / `HandleEvent` mantığı; **Render şablonla**
4. View: `internal/adapters/goui/views/pages/<ad>.goui.html`
5. Controller’da:
   ```go
   return p.RenderView("pages.<ad>", map[string]any{ /* … */ })
   ```
   Dosya yolu `views/pages/user_show.goui.html` → view adı `"pages.user_show"` (underscore → dot-path).
6. Ortak parçalar: `views/components/*` (`input`, `select`, `pagination`, `export_links`, …) ve
   `viewmodels.go` yardımcıları (`viewPagination`, `viewExportLinks`, `viewFieldError`, …)
7. Layout/shell zaten `layouts.shell` + partial’larda; sayfa yalnızca gövde HTML üretir
8. i18n: `dashboard.*` anahtarları (`panel_i18n.go` + `pkg/i18n/locales/tr.json` & `en.json`)
9. Export gerekiyorsa `export_routes.go` + `pkg/tabular`

Dev’de şablon hot-reload açıktır (`WatchForChanges`); prod’da gömülü `views` kullanılır.

## 8. Bootstrap

`internal/bootstrap` içinde bağımlılık grafiği `bootstrap.go` (Build orkestrasyonu) ve
`wire_*.go` dosyalarına bölünmüştür (`wire_infra`, `wire_repos`, `wire_authz`, …).
Yeni modül eklerken ilgili `wire_*.go` dosyasında:

- repo oluştur
- use-case oluştur
- handler / GoUI deps’e bağla

Yeni bağımlılık constructor injection ile eklenir; global singleton yok.
Testler için `internal/bootstrap/testkit` paket yorumuna bakın.

## 9. Test

- Domain entity unit test
- Application use-case (mock port — mockery)
- Handler / middleware gerekirse
- `go test ./...` ve CI yeşil

## 10. Doküman

- README özellik listesine bir satır
- Gerekirse `DOMAIN.md` tablosuna context satırı
- Public API ise OpenAPI + `/docs`
- Önerilir: `docs/map/<context>.md` (user/auth örneklerine bak)

## Yapılmayacaklar

- Domain’den Fiber, sqlc veya env okumak
- `pkg/` altında aggregate koymak (`pkg/rbac` yalnızca izin sabiti)
- İletişim formu ile alıcı CSV’yi karıştırmak (`contact` ≠ `pkg/recipients`)
- Secret’ı panel/DB’ye yazmak (SMS/ödeme kimlikleri env’de kalır)

Kontrol listesi dosyanın başındaki **Yazma checklist** ile aynıdır.