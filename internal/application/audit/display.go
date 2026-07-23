package audit

import (
	"fmt"
	"strings"
)

// formatChangeSummary, metadata'dan okunabilir eski→yeni özet üretir.
func formatChangeSummary(action string, meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	if old, newVal, ok := pair(meta, "old_email", "new_email"); ok {
		return fmt.Sprintf("%s → %s", old, newVal)
	}
	if old, newVal, ok := pair(meta, "old_name", "new_name"); ok {
		return fmt.Sprintf("%s → %s", old, newVal)
	}
	if old, newVal, ok := pair(meta, "old_phone", "new_phone"); ok {
		return phoneSummary(old, newVal)
	}
	if old, newVal, ok := pair(meta, "old_locale", "new_locale"); ok {
		return fmt.Sprintf("%s → %s", old, newVal)
	}
	if old, newVal, ok := pair(meta, "old_role", "new_role"); ok {
		return fmt.Sprintf("%s → %s", old, newVal)
	}
	switch action {
	case "user.registered":
		if email, _ := meta["email"].(string); email != "" {
			if name, _ := meta["name"].(string); name != "" {
				return fmt.Sprintf("%s (%s)", name, email)
			}
			return email
		}
	case "auth.login_succeeded":
		if provider, _ := meta["provider"].(string); provider != "" {
			if email, _ := meta["email"].(string); email != "" {
				return fmt.Sprintf("%s · %s", email, provider)
			}
			return provider
		}
	case "auth.login_failed":
		if reason, _ := meta["reason"].(string); reason != "" {
			if email, _ := meta["email"].(string); email != "" {
				return fmt.Sprintf("%s · %s", email, reason)
			}
			return reason
		}
	}
	return ""
}

func pair(meta map[string]any, oldKey, newKey string) (string, string, bool) {
	old, okOld := stringVal(meta[oldKey])
	newVal, okNew := stringVal(meta[newKey])
	return old, newVal, okOld && okNew
}

func stringVal(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func phoneSummary(old, newVal string) string {
	old = strings.TrimSpace(old)
	newVal = strings.TrimSpace(newVal)
	switch {
	case old == "" && newVal != "":
		return newVal
	case old != "" && newVal == "":
		return old + " → (kaldırıldı)"
	default:
		return fmt.Sprintf("%s → %s", old, newVal)
	}
}
