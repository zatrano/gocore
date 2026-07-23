# Map: Payment

3DS ödeme bounded context — Iyzico / Moka, işlem listesi, webhook.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/payment/` |
| Application | `internal/application/payment/` — **önce** `threeds.go` (`ThreeDSService`) |
| HTTP | `internal/adapters/http/handler/payment_handler.go` |
| GoUI | `internal/adapters/goui/controller_settings_payments_audit.go` (checkout, transactions) |
| Persistence | `adapters/persistence/postgres/payment_repository.go` |
| Altyapı | `internal/infrastructure/payment/iyzico`, `.../moka` |
| Wiring | `wire_user.go` (`ThreeDSSvc`); `wire_http.go`, `wire_goui.go` |

## Application (`ThreeDSService`)

| Dosya | Tipik iş |
|---|---|
| `threeds.go` | Facade: BinCheck, Initialize/Complete 3DS, List/Get, webhook/callback |
| `reconcile.go` | `ReconcileStale` — takılı işlemler |

## HTTP yüzeyleri

Kayıt: `internal/adapters/http/server.go` → `/api/v1/payments/...`

| Method | Path | Handler | İzin (özet) |
|---|---|---|---|
| POST | `/payments/bin-check` | BinCheck | `payments:charge` |
| POST | `/payments/3ds/initialize` | Initialize3DS | `payments:charge` |
| POST | `/payments/3ds/auth` | Complete3DS | `payments:charge` |
| POST | `/payments/calc-amount` | CalcPaymentAmount | `payments:charge` |
| GET | `/payments/transactions` | ListPayments | `payments:list` |
| GET | `/payments/transactions/:reference` | GetPayment | `payments:charge` veya `payments:list` |
| POST | `/payments/3ds/callback` | Callback3DS | public (sağlayıcı) |
| POST | `/payments/webhook/iyzico` | IyzicoWebhook | public (imza) |

## Panel yüzeyleri

| Path | Screen | Controller |
|---|---|---|
| `/dashboard/payments/checkout` | `checkout` | checkout |
| `/dashboard/payments/transactions` | `payments` | list |
| `/dashboard/payments/transactions/:reference` | `payment-show` | show |

Rota tablosu: `adapters/goui/routes.go` → `pageRoutes()`.

## İlişkili

- Aktif ödeme sağlayıcısı → [map/settings.md](settings.md) (`settings.Service`)
- Derinlik → [PAYMENT.md](../PAYMENT.md)
- Mimari → [DOMAIN.md](../DOMAIN.md)
