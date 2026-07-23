// Package i18n, çoklu dil (uluslararasılaştırma) desteği sağlar. Çevirmen
// (Translator) embed edilmiş JSON sözlüklerden mesajları yükler; anahtar
// bulunamazsa varsayılan dile, o da yoksa çağıranın verdiği fallback'e düşer.
//
// Anahtar konvansiyonu, domain hata kodlarıyla hizalıdır (ör. "user.not_found"),
// böylece RFC7807 Problem Details mesajları hata koduna göre yerelleştirilir.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

// Locale, bir dil kodudur (ör. "tr", "en").
type Locale string

// Default, hiçbir dil çözümlenemediğinde kullanılan güvenli varsayılandır.
const Default Locale = "tr"

// Translator, dil sözlüklerini tutar ve mesaj çevirir. Yükleme sonrası
// salt-okunurdur; eşzamanlı kullanım güvenlidir.
type Translator struct {
	catalog   map[Locale]map[string]string
	def       Locale
	supported []Locale
}

// New, verilen dosya sisteminden (dir altındaki "<locale>.json" dosyaları)
// desteklenen dilleri yükler. def, varsayılan dildir.
func New(fsys fs.FS, dir string, def Locale, supported []Locale) (*Translator, error) {
	if len(supported) == 0 {
		return nil, fmt.Errorf("i18n: en az bir dil desteklenmeli")
	}
	t := &Translator{
		catalog:   make(map[Locale]map[string]string, len(supported)),
		def:       def,
		supported: supported,
	}
	for _, loc := range supported {
		b, err := fs.ReadFile(fsys, path.Join(dir, string(loc)+".json"))
		if err != nil {
			return nil, fmt.Errorf("i18n: %q sözlüğü okunamadı: %w", loc, err)
		}
		m := make(map[string]string)
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("i18n: %q sözlüğü çözümlenemedi: %w", loc, err)
		}
		t.catalog[loc] = m
	}
	if _, ok := t.catalog[def]; !ok {
		return nil, fmt.Errorf("i18n: varsayılan dil %q desteklenenler arasında değil", def)
	}
	return t, nil
}

// T, verilen dilde anahtarı çevirir. Bulunamazsa sırasıyla varsayılan dile ve
// fallback'e düşer. args verilirse fmt.Sprintf ile biçimlenir.
func (t *Translator) T(loc Locale, key, fallback string, args ...any) string {
	if m, ok := t.catalog[loc]; ok {
		if msg, ok := m[key]; ok {
			return format(msg, args...)
		}
	}
	if loc != t.def {
		if m, ok := t.catalog[t.def]; ok {
			if msg, ok := m[key]; ok {
				return format(msg, args...)
			}
		}
	}
	return format(fallback, args...)
}

// DefaultLocale, çevirmenin varsayılan dilini döner.
func (t *Translator) DefaultLocale() Locale { return t.def }

// Supported, desteklenen dillerin listesini döner.
func (t *Translator) Supported() []Locale { return t.supported }

// Supports, dilin desteklenip desteklenmediğini döner.
func (t *Translator) Supports(l Locale) bool {
	for _, s := range t.supported {
		if s == l {
			return true
		}
	}
	return false
}

// Resolve, açık dil tercihini (ör. ?lang) ve Accept-Language başlığını
// değerlendirerek en uygun desteklenen dili seçer. Açık tercih önceliklidir.
func (t *Translator) Resolve(explicit, acceptLanguage string) Locale {
	if explicit != "" {
		l := Locale(strings.ToLower(strings.TrimSpace(explicit)))
		if t.Supports(l) {
			return l
		}
		if base, ok := baseLocale(l); ok && t.Supports(base) {
			return base
		}
	}
	return ParseAcceptLanguage(acceptLanguage, t.supported, t.def)
}

// baseLocale, "en-US" gibi bir kodun temel dilini ("en") döner.
func baseLocale(l Locale) (Locale, bool) {
	if i := strings.IndexByte(string(l), '-'); i >= 0 {
		return Locale(string(l)[:i]), true
	}
	return "", false
}

// format, "{0}", "{1}", ... yer tutucularını sırayla args ile değiştirir.
// Bilinçli olarak fmt.Sprintf KULLANMAZ: aksi halde `go vet` bu fonksiyonu (ve
// onu saran T'yi) bir printf-wrapper sanıp sabit-olmayan format string uyarısı
// verir. Yer tutucu tabanlı yaklaşım hem daha güvenli hem çeviri dostudur.
func format(s string, args ...any) string {
	if len(args) == 0 {
		return s
	}
	pairs := make([]string, 0, len(args)*2)
	for i, a := range args {
		pairs = append(pairs, "{"+strconv.Itoa(i)+"}", fmt.Sprint(a))
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// --- Context taşıyıcı ---

type ctxKey struct{}

type carrier struct {
	tr  *Translator
	loc Locale
}

// NewContext, çevirmeni ve çözümlenen dili context'e iliştirir (middleware kullanır).
func NewContext(ctx context.Context, tr *Translator, loc Locale) context.Context {
	return context.WithValue(ctx, ctxKey{}, carrier{tr: tr, loc: loc})
}

// LocaleFromContext, context'teki çözümlenmiş dili döner (yoksa Default).
func LocaleFromContext(ctx context.Context) Locale {
	if c, ok := ctx.Value(ctxKey{}).(carrier); ok {
		return c.loc
	}
	return Default
}

// T, context'teki çevirmen ve dili kullanarak anahtarı çevirir. Context'te
// çevirmen yoksa doğrudan fallback döner. Handler/render katmanının ana giriş
// noktasıdır; imza sadeliği için ctx üzerinden çalışır.
func T(ctx context.Context, key, fallback string, args ...any) string {
	if c, ok := ctx.Value(ctxKey{}).(carrier); ok && c.tr != nil {
		return c.tr.T(c.loc, key, fallback, args...)
	}
	return format(fallback, args...)
}
