package shared

import (
	"strings"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	"github.com/zatrano/gocore/pkg/recipients"
	"github.com/zatrano/gocore/pkg/validation"
)

// RecipientsFromParsed, recipients.Parse çıktısını bildirim alıcılarına çevirir.
func RecipientsFromParsed(list []recipients.Row) ([]appnotif.Recipient, error) {
	normalized, err := recipients.NormalizeAll(list)
	if err != nil {
		return nil, err
	}
	out := make([]appnotif.Recipient, 0, len(normalized))
	for _, row := range normalized {
		out = append(out, appnotif.Recipient{
			UserID: row.UserID, Phone: row.Phone, Email: row.Email, Locale: row.Locale,
		})
	}
	return out, nil
}

// RecipientsFromTextLines, satır satır alıcı listesini kanala göre ayrıştırır.
// inapp → e-posta, email → e-posta, sms → telefon.
// locale parametresi bilinçli olarak damgalanmaz: dil filtresi BulkSendCommand.Locale
// ve (varsa) kullanıcının tercih dili / CSV locale sütunu ile uygulanır.
func RecipientsFromTextLines(raw, locale string, channel appnotif.Channel) []appnotif.Recipient {
	_ = locale
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]appnotif.Recipient, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		r := appnotif.Recipient{}
		switch channel {
		case appnotif.ChannelEmail:
			email, err := validation.NormalizeEmail(line)
			if err != nil {
				continue
			}
			r.Email = email
		case appnotif.ChannelSMS:
			phone, err := validation.NormalizePhone(line)
			if err != nil {
				continue
			}
			r.Phone = phone
		case appnotif.ChannelInApp:
			email, err := validation.NormalizeEmail(line)
			if err != nil {
				continue
			}
			r.Email = email
		default:
			email, err := validation.NormalizeEmail(line)
			if err != nil {
				continue
			}
			r.Email = email
		}
		out = append(out, r)
	}
	return out
}

// BulkMessageContent, toplu bildirim form alanlarıdır.
type BulkMessageContent struct {
	Title, Body, HTMLBody, Locale string
}

// BuildBulkCommand, toplu gönderim komutunu oluşturur.
func BuildBulkCommand(channel appnotif.Channel, content BulkMessageContent, recipients []appnotif.Recipient) appnotif.BulkSendCommand {
	return appnotif.BulkSendCommand{
		Channel: channel,
		Content: appnotif.MessageContent{
			Title: content.Title, Body: content.Body, HTMLBody: content.HTMLBody,
		},
		Recipients: recipients,
		Locale:     content.Locale,
	}
}
