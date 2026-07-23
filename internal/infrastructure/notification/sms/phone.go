package sms

import "strings"

// normalizeTRMobile, Türkiye cep numarasını 5xxxxxxxxx formatına çevirir.
func normalizeTRMobile(to string) string {
	s := strings.TrimSpace(to)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.TrimPrefix(s, "+")
	s = strings.TrimPrefix(s, "00")
	if strings.HasPrefix(s, "90") && len(s) > 10 {
		s = s[2:]
	}
	s = strings.TrimPrefix(s, "0")
	if len(s) != 10 || s[0] != '5' {
		return ""
	}
	return s
}
