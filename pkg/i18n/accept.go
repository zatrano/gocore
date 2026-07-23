package i18n

import (
	"sort"
	"strconv"
	"strings"
)

// ParseAcceptLanguage, RFC 7231 Accept-Language başlığını q-değerlerine göre
// çözümler ve desteklenen diller arasından en yüksek öncelikli olanı seçer.
// "en-US" gibi bölgesel etiketler temel dile ("en") indirgenerek eşleştirilir.
// Eşleşme yoksa def döner.
func ParseAcceptLanguage(header string, supported []Locale, def Locale) Locale {
	header = strings.TrimSpace(header)
	if header == "" {
		return def
	}

	type pref struct {
		tag string
		q   float64
		idx int
	}
	parts := strings.Split(header, ",")
	prefs := make([]pref, 0, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segs := strings.Split(part, ";")
		tag := strings.ToLower(strings.TrimSpace(segs[0]))
		if tag == "" {
			continue
		}
		q := 1.0
		for _, s := range segs[1:] {
			s = strings.TrimSpace(s)
			if strings.HasPrefix(s, "q=") {
				if v, err := strconv.ParseFloat(strings.TrimPrefix(s, "q="), 64); err == nil {
					q = v
				}
			}
		}
		prefs = append(prefs, pref{tag: tag, q: q, idx: i})
	}

	// q azalan; eşitlikte başlıktaki sıra korunur (stabil).
	sort.SliceStable(prefs, func(a, b int) bool { return prefs[a].q > prefs[b].q })

	for _, p := range prefs {
		if p.q <= 0 {
			continue // q=0 => açıkça reddedilmiş
		}
		if p.tag == "*" {
			return def
		}
		cand := Locale(p.tag)
		for _, s := range supported {
			if s == cand {
				return s
			}
		}
		if base, ok := baseLocale(cand); ok {
			for _, s := range supported {
				if s == base {
					return s
				}
			}
		}
	}
	return def
}
