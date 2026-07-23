# Dağıtım

## Önkoşullar

- Go 1.25.12+
- PostgreSQL (production’da SSL: `DB_SSLMODE=require` veya üstü)
- Ortam dosyası: `.env` (asla commit etme; `.gitignore`’da)

## Build

```bash
go build -o bin/server ./cmd/server
go build -o bin/migrate ./cmd/migrate
go build -o bin/seed ./cmd/seed
```

CI aynı binary’leri derler (`.github/workflows/ci.yml`).

## Migration

```bash
./bin/migrate up          # veya: go run ./cmd/migrate up
./bin/migrate down 1      # isteğe bağlı geri adım
```

Şema: `migrations/000001_schema` + `000002_seed` (SQL seed / izin kataloğu).  
Admin kullanıcı için ayrıca:

```bash
./bin/seed
```

Canlıdan sonra yeni değişiklikler **incremental** migration dosyalarıyla eklenir; geliştirmede şema birleştirilebilir.

## Tek süreç dağıtımı (Redis yok)

GoCore tek süreçte çalışacak şekilde tasarlanmıştır; **Redis veya harici önbellek zorunlu değildir**.

Process içi bellek kullanan bileşenler:

- JWT refresh token store (`MemoryTokenStore`)
- Login guard ve IP rate limiter
- Authz `Resolver` (rol-izin önbelleği)
- Settings `Service` (platform ayarları önbelleği)

Bu yapı tek instance için yeterlidir. Birden fazla replica ile yatay ölçek planlandığında paylaşımlı store ve tutarlı invalidation eklenmelidir — mevcut sürümde kapsam dışı.

## Çalıştırma

```bash
./bin/server
```

Dinleme: `HTTP_HOST` + `HTTP_PORT` (varsayılan `0.0.0.0:8080`).  
Kapanış: `APP_SHUTDOWN_TIMEOUT`.

Health:

| Yol | Anlam |
|---|---|
| `/livez` | Süreç ayakta |
| `/readyz` | Bağımlılıklar hazır (DB vb.) |
| `/healthz` | Genel sağlık |

## Ortam grupları

| Prefix | Zorunluluk (production) |
|---|---|
| `APP_*`, `HTTP_*`, `DB_*` | Evet |
| `AUTH_JWT_SECRET` (≥32), MFA/encryption key’ler | Evet |
| `AUTH_EMAIL_LINK_BASE_URL` | Gerçek public URL |
| `OAUTH_*` | SSO kullanılacaksa |
| `SEC_TURNSTILE_*` | Evet (public formlar) |
| `SEC_TRUSTED_PROXIES` | Proxy arkasındaysa |
| `SMTP_*` | Gerçek e-posta için |
| `CONTACT_RECIPIENT_EMAIL` | Evet |
| `NOTIFY_*` | SMS kullanılacaksa |
| `PAYMENT_*` + field encryption key | Ödeme kullanılacaksa |

Tam liste: [`.env.example`](../.env.example). Uygulama boot’ta doğrular (fail-fast).

## Reverse proxy

- TLS’i proxy’de sonlandırın; `SEC_TRUSTED_PROXIES` ile gerçek client IP
- WebSocket: `/goui/ws` (panel UI) ve `/api/v1/ws` (genel canlı olay; bildirim) upgrade’e izin verin
- Body limiti: `HTTP_BODY_LIMIT_BYTES` / `SEC_MAX_UPLOAD_BYTES` ile uyumlu tutun

## Güvenlik checklist

- [ ] `.env` secret’ları güçlü ve rotasyona uygun
- [ ] `APP_ENV=production`
- [ ] DB SSL açık
- [ ] Turnstile, JWT, MFA ve payment encryption key set
- [ ] CORS: `HTTP_CORS_ALLOW_ORIGIN` daraltılmış
- [ ] Seed varsayılan admin parolası değiştirilmiş
- [ ] Iyzico/Moka callback ve webhook URL’leri panellerde kayıtlı

## Gözlem

- Log: `slog` (yapılandırılmış)
- Test/tarama: `go test -race`, `golangci-lint`, `govulncheck`, `gosec`
