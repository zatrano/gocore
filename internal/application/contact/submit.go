package contact

import (
	"context"
	"fmt"
	"html"
	"strings"

	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	domaincontact "github.com/zatrano/gocore/internal/domain/contact"
)

// SubmitCommand, iletişim formu gönderim girdisidir.
type SubmitCommand struct {
	Name      string
	Email     string
	Message   string
	Locale    string
	IP        string
	UserAgent string
}

// SubmitHandler, mesajı kaydeder ve alıcıya e-posta işini aynı transaction'da kuyruğa alır.
type SubmitHandler struct {
	repo      domaincontact.Repository
	outbox    appoutbox.Enqueuer
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	toEmail   string
}

// NewSubmitHandler, handler'ı kurar.
func NewSubmitHandler(
	repo domaincontact.Repository,
	outbox appoutbox.Enqueuer,
	publisher appshared.EventPublisher,
	tx appshared.TxManager,
	toEmail string,
) *SubmitHandler {
	return &SubmitHandler{repo: repo, outbox: outbox, publisher: publisher, tx: tx, toEmail: toEmail}
}

// Handle, iletişim mesajını kalıcılaştırır.
func (h *SubmitHandler) Handle(ctx context.Context, cmd SubmitCommand) (View, error) {
	msg, err := domaincontact.Submit(cmd.Name, cmd.Email, cmd.Message, cmd.Locale, cmd.IP, cmd.UserAgent)
	if err != nil {
		return View{}, err
	}

	err = h.tx.WithinTx(ctx, func(ctx context.Context) error {
		msg.MarkQueued()
		if err := h.repo.Save(ctx, msg); err != nil {
			return err
		}
		if h.toEmail != "" {
			subject := fmt.Sprintf("İletişim formu: %s", msg.Name())
			text := fmt.Sprintf("Ad: %s\nE-posta: %s\nDil: %s\nIP: %s\n\n%s",
				msg.Name(), msg.Email().String(), msg.Locale(), msg.IP(), msg.Body())
			htmlBody := contactEmailHTML(msg)
			job, err := appoutbox.NewJob(
				appoutbox.KindEmailSend,
				"contact_message",
				msg.ID().String(),
				"contact-email:"+msg.ID().String(),
				appoutbox.EmailPayload{
					To:       []string{h.toEmail},
					Subject:  subject,
					TextBody: text,
					HTMLBody: htmlBody,
					ReplyTo:  msg.Email().String(),
				},
			)
			if err != nil {
				return err
			}
			if err := h.outbox.Enqueue(ctx, job); err != nil {
				return err
			}
		}
		return h.publisher.Publish(ctx, msg.PullEvents()...)
	})
	if err != nil {
		return View{}, err
	}
	return View{ID: msg.ID().String()}, nil
}

func contactEmailHTML(msg *domaincontact.Message) string {
	body := html.EscapeString(msg.Body())
	body = strings.ReplaceAll(body, "\n", "<br>")
	return fmt.Sprintf(
		"<p><strong>Ad:</strong> %s</p><p><strong>E-posta:</strong> %s</p><p><strong>Dil:</strong> %s</p><p><strong>IP:</strong> %s</p><hr><p>%s</p>",
		html.EscapeString(msg.Name()),
		html.EscapeString(msg.Email().String()),
		html.EscapeString(msg.Locale()),
		html.EscapeString(msg.IP()),
		body,
	)
}
