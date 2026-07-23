package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// SMTPMailer, genel SMTP üzerinden e-posta gönderir.
type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	tlsMode  string // none | starttls | tls
	timeout  time.Duration
}

// NewSMTPMailer, SMTP yapılandırmasından mailer kurar.
func NewSMTPMailer(cfg config.SMTP) *SMTPMailer {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.TLSMode))
	if mode == "" {
		mode = "starttls"
	}
	return &SMTPMailer{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password.Value(),
		from:     cfg.From,
		tlsMode:  mode,
		timeout:  timeout,
	}
}

// NewMailer, SMTP yapılandırılmışsa SMTPMailer, aksi halde LogMailer döner.
func NewMailer(cfg config.SMTP, log *slog.Logger) appshared.Mailer {
	if cfg.Configured() {
		return NewSMTPMailer(cfg)
	}
	return NewLogMailer(log)
}

// Send, e-postayı SMTP ile iletir.
func (m *SMTPMailer) Send(ctx context.Context, e appshared.Email) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(e.To) == 0 {
		return fmt.Errorf("smtp: alıcı gerekli")
	}
	from := e.From
	if from == "" {
		from = m.from
	}
	if from == "" {
		return fmt.Errorf("smtp: from adresi gerekli")
	}

	msg := buildMIME(from, e)
	addr := fmt.Sprintf("%s:%d", m.host, m.port)

	netDialer := &net.Dialer{Timeout: m.timeout}
	var (
		conn net.Conn
		err  error
	)
	switch m.tlsMode {
	case "tls":
		tlsDialer := &tls.Dialer{
			NetDialer: netDialer,
			Config:    &tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	default:
		conn, err = netDialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(m.timeout))

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if m.tlsMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		}
	}

	if m.username != "" {
		auth := smtp.PlainAuth("", m.username, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, to := range e.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return client.Quit()
}

func buildMIME(from string, e appshared.Email) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(e.To, ", ") + "\r\n")
	if e.ReplyTo != "" {
		b.WriteString("Reply-To: " + e.ReplyTo + "\r\n")
	}
	b.WriteString("Subject: " + sanitizeHeader(e.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	text := e.TextBody
	html := e.HTMLBody
	switch {
	case html != "" && text != "":
		boundary := "zatrano-boundary"
		b.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(text + "\r\n")
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(html + "\r\n")
		b.WriteString("--" + boundary + "--\r\n")
	case html != "":
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(html)
	default:
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(text)
	}
	return []byte(b.String())
}

func sanitizeHeader(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, s)
}
