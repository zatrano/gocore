// Package shared, HTTP ve GoUI web adapter'larının ortak sabitlerini ve
// yardımcılarını barındırır. Adapter'lar arası çapraz bağımlılığı önler.
package shared

// Fiber Locals ve cookie anahtarları — tüm adapter'lar aynı sabitleri kullanır.
const (
	LocalUserID = "user_id"
	LocalRole   = "role"
	LocalEmail  = "email"
	LocalLocale = "locale"
	LangCookie  = "lang"
)
