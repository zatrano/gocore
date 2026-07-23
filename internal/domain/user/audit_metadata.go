package user

import "github.com/zatrano/gocore/internal/domain/shared"

// AuditMetadata, domain event'ten denetim kaydı metadata'sını çıkarır.
// Eski/yeni değer çiftleri varsa old_* / new_* anahtarlarıyla döner.
func AuditMetadata(e shared.DomainEvent) map[string]any {
	switch ev := e.(type) {
	case RegisteredEvent:
		return map[string]any{
			"email":            ev.Email,
			"name":             ev.Name,
			"preferred_locale": ev.PreferredLocale,
		}
	case EmailChangedEvent:
		return map[string]any{"old_email": ev.OldEmail, "new_email": ev.NewEmail}
	case NameChangedEvent:
		return map[string]any{"old_name": ev.OldName, "new_name": ev.NewName}
	case PhoneChangedEvent:
		return map[string]any{"old_phone": ev.OldPhone, "new_phone": ev.NewPhone}
	case LocaleChangedEvent:
		return map[string]any{"old_locale": ev.OldLocale, "new_locale": ev.NewLocale}
	case RoleChangedEvent:
		return map[string]any{"old_role": ev.OldRole, "new_role": ev.NewRole}
	default:
		return nil
	}
}
