package http

import "strings"

// splitCSV, virgülle ayrılmış bir değeri temizlenmiş dilime çevirir.
// Boş girdi için "*" (tüm origin'ler) döner.
func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return []string{"*"}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
