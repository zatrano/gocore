package audit

import (
	"encoding/json"
	"strings"

	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
)

// MetadataFromDomainEvent, event türüne göre güvenli metadata üretir.
func MetadataFromDomainEvent(p appoutbox.DomainEventPayload) map[string]any {
	base := map[string]any{
		"event_id":   p.EventID,
		"event_name": p.EventName,
	}
	var raw map[string]any
	_ = json.Unmarshal(p.Data, &raw)
	safe := map[string]any{}
	for k, v := range raw {
		switch k {
		case "BaseEvent", "ID", "AggregateID_", "id", "At":
			continue
		case "Email", "email":
			safe["email"] = v
		case "Name", "name":
			if _, exists := safe["name"]; !exists {
				safe["name"] = v
			}
		case "PreferredLocale", "preferred_locale":
			safe["preferred_locale"] = v
		case "OldEmail", "old_email":
			safe["old_email"] = v
		case "NewEmail", "new_email":
			safe["new_email"] = v
		case "OldName", "old_name":
			safe["old_name"] = v
		case "NewName", "new_name":
			safe["new_name"] = v
		case "OldPhone", "old_phone":
			safe["old_phone"] = v
		case "NewPhone", "new_phone":
			safe["new_phone"] = v
		case "OldRole", "old_role":
			safe["old_role"] = v
		case "NewRole", "new_role":
			safe["new_role"] = v
		case "OldLocale", "old_locale":
			safe["old_locale"] = v
		case "NewLocale", "new_locale":
			safe["new_locale"] = v
		default:
			if k != "" && k[0] >= 'A' && k[0] <= 'Z' {
				safe[toSnake(k)] = v
			} else if isSafeKey(k) {
				safe[k] = v
			}
		}
	}
	return RedactMetadata(merge(base, safe))
}

func isSafeKey(k string) bool {
	switch k {
	case "provider", "old_provider", "new_provider", "reference", "status", "stage",
		"amount", "currency", "channel", "accepted", "total", "failed", "invalid",
		"mime", "size", "count", "key", "reason", "purpose":
		return true
	default:
		return false
	}
}

func toSnake(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteByte(c + 'a' - 'A')
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func merge(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
