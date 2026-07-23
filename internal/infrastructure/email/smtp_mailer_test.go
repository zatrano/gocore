package email

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

func TestBuildMIME_ReplyToAndMultipart(t *testing.T) {
	msg := string(buildMIME("noreply@example.com", appshared.Email{
		To:       []string{"a@b.com"},
		Subject:  "Hi\nBad",
		TextBody: "plain",
		HTMLBody: "<b>html</b>",
		ReplyTo:  "user@example.com",
	}))
	if !strings.Contains(msg, "Reply-To: user@example.com") {
		t.Fatalf("missing reply-to: %s", msg)
	}
	if strings.Contains(msg, "Hi\nBad") {
		t.Fatalf("subject newline not sanitized")
	}
	if !strings.Contains(msg, "multipart/alternative") {
		t.Fatalf("expected multipart")
	}
}

func TestNewMailer_FallbackLog(t *testing.T) {
	m := NewMailer(config.SMTP{}, slog.Default())
	if _, ok := m.(*LogMailer); !ok {
		t.Fatalf("expected LogMailer, got %T", m)
	}
}

func TestNewSMTPMailer_Defaults(t *testing.T) {
	m := NewSMTPMailer(config.SMTP{Host: "smtp.example.com", From: "a@b.com", Timeout: 0})
	if m.timeout != 15*time.Second {
		t.Fatalf("timeout %v", m.timeout)
	}
	if m.tlsMode != "starttls" {
		t.Fatalf("tls %q", m.tlsMode)
	}
}
