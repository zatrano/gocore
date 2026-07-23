package i18n

import "embed"

//go:embed locales/*.json
var localesFS embed.FS

// NewFromEmbedded, ikili dosyaya gömülü sözlüklerden bir çevirmen oluşturur.
// Harici dosya bağımlılığı olmadan tek binary dağıtımını mümkün kılar.
func NewFromEmbedded(def Locale, supported []Locale) (*Translator, error) {
	return New(localesFS, "locales", def, supported)
}
