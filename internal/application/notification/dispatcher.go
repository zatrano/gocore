package notification

import (
	"context"
	"strings"
)

// SendCommand, merkezi bildirim gönderme girdisidir. Çağıran yalnızca türü
// (Channel) ve o türe ait alanları doldurur; başlık/içerik i18n anahtarıdır.
//
// Tür bazlı zorunlu alanlar:
//   - inapp: Email, TitleKey, ContentKey
//   - sms:   Phone, ContentKey
//   - email: Email, TitleKey, ContentKey (+ opsiyonel HTMLContentKey)
type SendCommand struct {
	Channel Channel

	// Alıcı bilgisi (türüne göre biri kullanılır).
	UserID string // inapp
	Phone  string // sms
	Email  string // email

	// İçerik: i18n anahtarları + parametreler.
	TitleKey      string
	ContentKey    string
	TitleFallback string // anahtar yoksa kullanılacak metin
	BodyFallback  string
	// HTMLContentKey / HTMLBodyFallback yalnızca email için; boşsa HTML gövde gönderilmez.
	HTMLContentKey   string
	HTMLBodyFallback string
	Args             []any

	// Locale, boşsa Dispatcher'ın varsayılan dili kullanılır.
	Locale string
}

func (c SendCommand) validate() error {
	if c.ContentKey == "" && c.BodyFallback == "" {
		return ErrContentRequired
	}
	switch c.Channel {
	case ChannelInApp:
		if strings.TrimSpace(c.UserID) == "" {
			return ErrRecipientRequired
		}
		if c.TitleKey == "" && c.TitleFallback == "" {
			return ErrTitleRequired
		}
	case ChannelSMS:
		if strings.TrimSpace(c.Phone) == "" {
			return ErrPhoneRequired
		}
	case ChannelEmail:
		if strings.TrimSpace(c.Email) == "" {
			return ErrEmailRequired
		}
		if c.TitleKey == "" && c.TitleFallback == "" {
			return ErrTitleRequired
		}
	default:
		return ErrUnsupportedChannel
	}
	return nil
}

// Dispatcher, merkezi bildirim gönderme servisidir (tüm sistemin tek giriş
// noktası). Kanal adaptörlerini bir kayıt (registry) olarak tutar ve türe göre
// yönlendirir.
type Dispatcher struct {
	translator    Translator
	senders       map[Channel]ChannelSender
	defaultLocale string
}

// NewDispatcher, çevirmen, varsayılan dil ve kanal adaptörleriyle Dispatcher
// kurar. Her sender kendi Channel()'ına göre kaydedilir.
func NewDispatcher(translator Translator, defaultLocale string, senders ...ChannelSender) *Dispatcher {
	m := make(map[Channel]ChannelSender, len(senders))
	for _, s := range senders {
		m[s.Channel()] = s
	}
	return &Dispatcher{translator: translator, senders: m, defaultLocale: defaultLocale}
}

// Send, komutu doğrular, içeriği isteğe/alıcıya ait dile render eder ve ilgili
// kanal adaptörüne iletir.
func (d *Dispatcher) Send(ctx context.Context, cmd SendCommand) error {
	if err := cmd.validate(); err != nil {
		return err
	}
	sender, ok := d.senders[cmd.Channel]
	if !ok {
		return ErrUnsupportedChannel
	}

	locale := cmd.Locale
	if locale == "" {
		locale = d.defaultLocale
	}

	title := cmd.TitleFallback
	if cmd.TitleKey != "" {
		title = d.translator.T(locale, cmd.TitleKey, cmd.TitleFallback, cmd.Args...)
	}
	content := cmd.BodyFallback
	if cmd.ContentKey != "" {
		content = d.translator.T(locale, cmd.ContentKey, cmd.BodyFallback, cmd.Args...)
	}
	htmlContent := ""
	if cmd.HTMLContentKey != "" || cmd.HTMLBodyFallback != "" {
		htmlContent = cmd.HTMLBodyFallback
		if cmd.HTMLContentKey != "" {
			htmlContent = d.translator.T(locale, cmd.HTMLContentKey, cmd.HTMLBodyFallback, cmd.Args...)
		}
	}

	return sender.Send(ctx, RenderedMessage{
		Channel:         cmd.Channel,
		RecipientUserID: cmd.UserID,
		Phone:           cmd.Phone,
		Email:           cmd.Email,
		Title:           title,
		Content:         content,
		HTMLContent:     htmlContent,
		Locale:          locale,
	})
}
