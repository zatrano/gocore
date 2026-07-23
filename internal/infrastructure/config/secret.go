package config

// Secret, hassas string değerleri temsil eder. Loglama, JSON serileştirme veya
// fmt ile yazdırma sırasında değeri sızdırmamak için maskelenir. Gerçek değere
// yalnızca Value() ile erişilir.
type Secret string

// String, fmt.Stringer arayüzünü sağlar ve değeri maskeler.
func (s Secret) String() string { return "***REDACTED***" }

// GoString, %#v formatında da maskeleme yapar.
func (s Secret) GoString() string { return "***REDACTED***" }

// MarshalJSON, JSON'a serileştirirken değeri maskeler.
func (s Secret) MarshalJSON() ([]byte, error) {
	return []byte(`"***REDACTED***"`), nil
}

// Value, gerçek gizli değeri döner. Yalnızca bağlantı kurma gibi
// zorunlu yerlerde çağrılmalıdır.
func (s Secret) Value() string { return string(s) }

// IsEmpty, secret'ın boş olup olmadığını döner.
func (s Secret) IsEmpty() bool { return string(s) == "" }
