# Map: Audit

Denetim kayıtları (application servisi; ayrı domain aggregate yok).

## Konumlar

| Katman | Yol |
|---|---|
| Application | `internal/application/audit/` — **önce** `service.go` |
| HTTP | `adapters/http/handler/audit_handler.go` |
| GoUI | `controller_settings_payments_audit.go` |
| Persistence | `audit_repository.go` |
| Wiring | `wire_user` → `auditService` |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: List, Get |
| `query.go` / `get.go` | CQRS handler’lar |

## Yüzeyler

- `GET /api/v1/audit/logs`, `/api/v1/audit/logs/:id`
- Panel: `/dashboard/audit/logs`
