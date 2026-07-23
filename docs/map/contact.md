# Map: Contact

İletişim formu bounded context.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/contact/` |
| Application | `internal/application/contact/` — **önce** `service.go` |
| HTTP | `internal/adapters/http/handler/contact_handler.go` |
| GoUI | `controller_public_auth.go` (submit), `controller_contacts.go` (inbox) |
| Persistence | `adapters/persistence/postgres/contact_repository.go` |
| Wiring | `wire_user.go` → `contactService`; `wire_http` / `wire_goui` |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: Submit, List, Get, MarkRead |
| `submit.go` | `SubmitHandler` |
| `query.go` | List / Get / MarkRead handler’ları |

## Yüzeyler

- Public: `POST /api/v1/contact`, panel `/contact`
- Admin: `/api/v1/contacts`, `/dashboard/contacts`
