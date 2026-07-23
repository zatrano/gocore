# Map: Notification

Bildirim (inbox + manuel/toplu gönderim) bounded context.

## Konumlar

| Katman | Yol |
|---|---|
| Domain | `internal/domain/notification/` |
| Application | `internal/application/notification/` — **önce** `service.go` |
| HTTP | `adapters/http/handler/notification_handler.go` |
| GoUI | inbox: `controller_account_users.go`; gönderim: `controller_rbac_notifications.go` |
| Persistence | `notification_repository.go` |
| Wiring | handlers `wire_workers`; `notifService` `wire_user`; HTTP/GoUI wire |

## Application

| Dosya | İş |
|---|---|
| `service.go` | Facade: List, MarkRead, UnreadCount, SendOne/Bulk… |
| `query.go` / `command.go` | Inbox CQRS |
| `bulk.go` | `ManualSender` |
| `dispatcher` ilgili | Kanal gönderimi |

## Not

Realtime hub hâlâ `UnreadCountHandler.Handle` callback’i kullanabilir; transport `Notifications.UnreadCount` çağırır — aynı handler.
