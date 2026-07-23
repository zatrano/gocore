// Package notification (application), merkezi bildirim sisteminin use-case
// katmanıdır. Tek giriş noktası Dispatcher'dır: çağıran taraf yalnızca türü
// (kanal) ve o türe ait parametreleri verir; Dispatcher içeriği i18n ile render
// edip ilgili kanal adaptörüne yönlendirir.
package notification

// Channel, bir bildirimin gönderileceği kanaldır (tür). Değerler dilden bağımsız
// İngilizce kimliklerdir; kullanıcıya dönük metinler i18n ile ayrıca çevrilir.
type Channel string

const (
	// ChannelInApp, uygulama içi kalıcı bildirim.
	ChannelInApp Channel = "inapp"
	// ChannelSMS, SMS bildirimi.
	ChannelSMS Channel = "sms"
	// ChannelEmail, e-posta bildirimi.
	ChannelEmail Channel = "email"
)

// ParseChannel, string bir türü Channel'a çevirir ve doğrular.
func ParseChannel(s string) (Channel, error) {
	switch Channel(s) {
	case ChannelInApp, ChannelSMS, ChannelEmail:
		return Channel(s), nil
	default:
		return "", ErrUnsupportedChannel
	}
}

func (c Channel) String() string { return string(c) }
