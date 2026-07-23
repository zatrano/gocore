# Doğrulama (validation)

## Stack

- [go-playground/validator/v10](https://github.com/go-playground/validator)
- Ortak kayıt: `pkg/validation.Register`
- Alan mesajları (i18n): `pkg/validation.FieldMessage` + `pkg/i18n/locales/{tr,en}.json`

HTTP ve GoUI aynı validator örneğini / mesaj çözümlemesini kullanır; hata yanıtları locale’e göre çevrilir.

## Özel tag’ler

`pkg/validation` içinde kayıtlı:

| Tag | Anlam |
|---|---|
| `phone` | Opsiyonel telefon; doluysa format kontrolü |
| `phone_required` | Zorunlu telefon |

Standart tag’ler (`required`, `email`, `min`, `max`, `uuid`, …) doğrudan kullanılır.

## Katmanlar

1. **Transport** — handler / GoUI form bind + struct tag
2. **Application** — iş kuralı (ör. son admin silinemez)
3. **Domain** — entity factory / VO invariant

Transport hataları kullanıcıya alan bazlı mesaj döner. Domain hataları problem/flash olarak map edilir.

## i18n

- Varsayılan / desteklenen: `I18N_DEFAULT_LOCALE`, `I18N_SUPPORTED`
- Çözüm sırası (özet): query `lang` → cookie → `Accept-Language` → kullanıcı `preferred_locale` → default
- Yeni validation mesajı eklerken **tr ve en** locale dosyalarına aynı anahtarı yazın
- Parite testi: `pkg/i18n/locale_parity_test.go`

## API hata biçimi

JSON API problem detaylarında alan hataları i18n metinleriyle gelir.  
Panelde flash / SweetAlert ile gösterilir.

## Upload / tabular

- MIME allowlist: `SEC_ALLOWED_UPLOAD_MIME` (görsel, PDF, CSV, Excel)
- Boyut: `SEC_MAX_UPLOAD_BYTES`
- Bulk alıcı parse: `pkg/recipients` (satır doğrulama)
- Export: `pkg/tabular`

## İlgili kod

- `pkg/validation`
- `internal/adapters/shared/validate.go`
- `internal/adapters/http/middleware/i18n.go`
