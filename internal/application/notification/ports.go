package notification

import "context"

// Translator, i18n render portudur. *i18n.Translator'ı sarmalayan bir adaptör
// bu arayüzü karşılar (Dependency Inversion; use-case pkg/i18n'e doğrudan bağlı
// kalmaz, test edilebilir olur).
type Translator interface {
	T(locale, key, fallback string, args ...any) string
}

// RenderedMessage, i18n render sonrası nihai (hazır metin) bildirimdir. Kanal
// adaptörleri yalnızca bu hazır veriyi alır.
type RenderedMessage struct {
	Channel         Channel
	RecipientUserID string // bildirim (in-app) için alıcı
	Phone           string // sms için
	Email           string // email için
	Title           string
	Content         string // düz metin (email TextBody, sms/inapp gövde)
	HTMLContent     string // opsiyonel; yalnızca email HTMLBody
	Locale          string
}

// ChannelSender, tek bir kanalı (bildirim/sms/email) gönderen porttur.
// İmplementasyonlar infrastructure katmanındadır.
type ChannelSender interface {
	// Channel, bu gönderenin sorumlu olduğu kanalı döner.
	Channel() Channel
	// Send, render edilmiş mesajı ilgili kanala iletir.
	Send(ctx context.Context, msg RenderedMessage) error
}

// UserContact, toplu/broadcast gönderimde kullanıcılardan türetilen iletişim bilgisidir.
type UserContact struct {
	ID     string
	Email  string
	Phone  string
	Locale string
}

// UserDirectory, aktif kullanıcı listesini sayfa sayfa okur (broadcast için).
type UserDirectory interface {
	ListActiveContacts(ctx context.Context, page int, limit int) (items []UserContact, hasMore bool, err error)
}
