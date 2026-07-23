# Map: Upload

Güvenli çoklu dosya yükleme bounded context.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/upload/` |
| Application | `internal/application/upload/` — **önce** `service.go` |
| HTTP | `internal/adapters/http/handler/upload_handler.go` |
| GoUI | `routes.go` (upload middleware); bildirim toplu yükleme: `controller_rbac_notifications.go` |
| Altyapı | `pkg/safefs`, `internal/infrastructure/storage` |
| Wiring | `wire_infra.go`, `wire_http.go`, `wire_goui.go` |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: `UploadBatch` — doğrulama, virüs taraması, depolama |
| `service_test.go` | Birim testleri |

## HTTP yüzeyleri

| Method | Path | Handler | İzin |
|---|---|---|---|
| POST | `/api/v1/uploads/` | Upload | `uploads:create` |

GoUI statik dosya yolu: `gouiupload.UploadPath` / `FilesPrefix` — aynı izin (`uploads:create` veya bildirim gönderimi).

## Panel yüzeyleri

| Path | Screen |
|---|---|
| `/dashboard/uploads` | `uploads` |
| `/dashboard/notifications/bulk/upload` | `notification-upload` |

## İlişkili

- Boyut limitleri → `SEC_MAX_UPLOAD_BYTES`, `HTTP_BODY_LIMIT_BYTES`
- Mimari → [DOMAIN.md](../DOMAIN.md)
