package shared

import (
	"testing"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
)

func TestRecipientsFromTextLines_byChannel(t *testing.T) {
	raw := "a@b.com\nuser@example.com\n+905551112233\n"
	email := RecipientsFromTextLines(raw, "tr", appnotif.ChannelEmail)
	if len(email) != 2 || email[0].Email != "a@b.com" || email[1].Email != "user@example.com" {
		t.Fatalf("email parse: %+v", email)
	}
	sms := RecipientsFromTextLines("+905551112233\n+905559998877", "en", appnotif.ChannelSMS)
	if len(sms) != 2 || sms[0].Phone != "+905551112233" || sms[0].Locale != "" {
		t.Fatalf("sms parse: %+v", sms)
	}
	inappEmail := RecipientsFromTextLines("a@b.com\nuser@example.com", "tr", appnotif.ChannelInApp)
	if len(inappEmail) != 2 || inappEmail[0].Email != "a@b.com" || inappEmail[1].Email != "user@example.com" {
		t.Fatalf("inapp email parse: %+v", inappEmail)
	}
	skipped := RecipientsFromTextLines("not-an-email\nuser-1", "tr", appnotif.ChannelInApp)
	if len(skipped) != 0 {
		t.Fatalf("geçersiz inapp satırları atlanmalı: %+v", skipped)
	}
}
