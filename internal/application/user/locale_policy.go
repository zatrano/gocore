package user

import (
	"strings"

	duser "github.com/zatrano/gocore/internal/domain/user"
)

// LocalePolicy, desteklenen dilleri doğrular. Application katmanında config'ten
// enjekte edilir; domain yalnızca format doğrulaması yapar.
type LocalePolicy struct {
	Default   string
	Supported map[string]struct{}
}

// NewLocalePolicy, desteklenen dil listesinden politika oluşturur.
func NewLocalePolicy(defaultLocale string, supported []string) LocalePolicy {
	set := make(map[string]struct{}, len(supported))
	for _, s := range supported {
		set[strings.ToLower(strings.TrimSpace(s))] = struct{}{}
	}
	return LocalePolicy{Default: defaultLocale, Supported: set}
}

// Resolve, boş girdide varsayılanı; dolu girdide desteklenen dili döner.
func (p LocalePolicy) Resolve(raw string) (duser.PreferredLocale, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return duser.ParsePreferredLocale(p.Default)
	}
	loc, err := duser.ParsePreferredLocale(raw)
	if err != nil {
		return duser.PreferredLocale{}, err
	}
	if _, ok := p.Supported[loc.String()]; !ok {
		return duser.PreferredLocale{}, duser.ErrUnsupportedLocale
	}
	return loc, nil
}
