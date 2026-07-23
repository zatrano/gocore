# Ödeme (3DS)

## Sağlayıcılar

Aktif sağlayıcı panelden seçilir: **Ayarlar → Ödeme** (`notifications:settings` izni).  
Kimlik bilgileri **yalnızca env**’dedir; panel secret tutmaz.

| Sağlayıcı | Env öneki | Varsayılan base |
|---|---|---|
| Iyzico | `PAYMENT_IYZICO_*` | sandbox-api.iyzipay.com |
| Moka United | `PAYMENT_MOKA_*` | service.refmokaunited.com |

Ortak:

| Değişken | Açıklama |
|---|---|
| `PAYMENT_CALLBACK_URL` | 3DS banka dönüşü; boşsa `{AUTH_EMAIL_LINK_BASE_URL}/api/v1/payments/3ds/callback` |
| `PAYMENT_FIELD_ENCRYPTION_KEY` | Kart alanı AES-256-GCM (`openssl rand -base64 32`); production zorunlu |

## Akış

```
BIN check → 3DS initialize → banka sayfası → callback → 3DS auth (tamamlama)
```

1. **BIN** — kart ilk 6–8 hane; taksit seçenekleri
2. **Initialize** — sağlayıcıya 3DS başlatma; HTML/redirect döner
3. **Callback** — banka POST; sunucu auth’u tamamlar
4. **Auth / complete** — işlem kaydı güncellenir (başarılı / başarısız)

İstemci IP sunucudan alınır (`c.IP()`); body’den client IP kabul edilmez.

## HTTP uçları

### Public (imza / banka)

| Metot | API | Web |
|---|---|---|
| POST | `/api/v1/payments/3ds/callback` | `/payments/3ds/callback` |
| POST | `/api/v1/payments/webhook/iyzico` | `/payments/webhook/iyzico` |

Iyzico webhook: `X-IYZ-SIGNATURE-V3` → `PAYMENT_IYZICO_SECRET_KEY`.

### Korumalı (`payments:charge` / `payments:list`)

| Metot | Yol | İzin |
|---|---|---|
| POST | `/api/v1/payments/bin-check` | charge |
| POST | `/api/v1/payments/3ds/initialize` | charge |
| POST | `/api/v1/payments/3ds/auth` | charge |
| POST | `/api/v1/payments/calc-amount` | charge |
| GET | `/api/v1/payments/transactions` | list |
| GET | `/api/v1/payments/transactions/:reference` | charge veya list |

Panel: `/dashboard/payments/checkout`, `/dashboard/payments/transactions`.

## Dayanıklılık

- Outbox / retry ve reconciliation altyapısı ödeme use-case’leriyle entegre
- Hassas kart alanları `pkg/fieldenc` + `PAYMENT_FIELD_ENCRYPTION_KEY`
- Idempotency middleware mutasyonlarda (callback politikası ayrı)

## Ayarlar API

| Metot | Yol |
|---|---|
| GET/POST | `/api/v1/settings/payment` |
| GET/PATCH | `/api/v1/settings/payment/:provider` |

## Kod

- `internal/application/payment`
- `internal/infrastructure/payment`
- `internal/adapters/http/handler/payment_handler.go`
- `internal/adapters/goui` (checkout / transactions ekranları)
