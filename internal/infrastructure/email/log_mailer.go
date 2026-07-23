// Package email, appshared.Mailer portunun implementasyonlarını içerir.
package email

import (
	"context"
	"log/slog"

	appshared "github.com/zatrano/gocore/internal/application/shared"
)

// LogMailer, e-postaları göndermek yerine loglar. Geliştirme/test için idealdir;
// üretimde SMTP/SES/SendGrid adaptörüyle değiştirilir (port aynı kalır).
type LogMailer struct {
	log *slog.Logger
}

// NewLogMailer, mailer'ı kurar.
func NewLogMailer(log *slog.Logger) *LogMailer { return &LogMailer{log: log} }

// Send, e-postayı loglar.
func (m *LogMailer) Send(ctx context.Context, e appshared.Email) error {
	m.log.InfoContext(ctx, "email gönderildi (log-only)",
		slog.Any("to", e.To),
		slog.String("subject", e.Subject),
	)
	return nil
}
