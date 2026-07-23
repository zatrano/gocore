# Map: Settings

Platform ayarları — SMS / ödeme sağlayıcı kayıtları ve aktif seçim.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/settings/` |
| Application | `internal/application/settings/` — **önce** `service.go` |
| HTTP | `internal/adapters/http/handler/settings_handler.go` |
| GoUI | `internal/adapters/goui/controller_settings_payments_audit.go` |
| Persistence | `adapters/persistence/postgres/settings_repository.go` |
| Wiring | `wire_user.go` (`settingsService`); `wire_http.go`, `wire_goui.go` |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: SMS/ödeme sağlayıcı CRUD, aktif sağlayıcı, bellek içi önbellek |
| `dto.go` | Dışarıya giden view/DTO |

Tek süreç dağıtımında `Service` ayarları process belleğinde önbellekler; yatay ölçekte paylaşımlı store gerekir (şimdilik yok).

## HTTP yüzeyleri

Kayıt: `server.go` → `/api/v1/settings/...` (izin: `notifications:settings`)

| Method | Path | Handler |
|---|---|---|
| GET/POST | `/settings/sms` | ListSMS / CreateSMS |
| GET/PATCH | `/settings/sms/:provider` | Get / Update SMS |
| GET/POST | `/settings/payment` | ListPayment / CreatePayment |
| GET/PATCH | `/settings/payment/:provider` | Get / Update payment |

## Panel yüzeyleri

| Path | Screen |
|---|---|
| `/dashboard/settings/sms` | `sms-settings` |
| `/dashboard/settings/sms/:provider` | `sms-provider` |
| `/dashboard/settings/payment` | `payment-settings` |
| `/dashboard/settings/payment/:provider` | `payment-provider` |

## İlişkili

- Ödeme akışı aktif sağlayıcıyı buradan okur → [map/payment.md](payment.md)
- Bildirim SMS kanalı → [map/notification.md](notification.md)
