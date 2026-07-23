package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// fakeTranslator, anahtarı olduğu gibi döner (render'ı gözlemlemek için).
type fakeTranslator struct{}

func (fakeTranslator) T(_ /*locale*/, key, fallback string, _ ...any) string {
	if key != "" {
		return key
	}
	return fallback
}

// recordingSender, tek bir kanala gelen son mesajı kaydeder.
type recordingSender struct {
	ch   Channel
	last RenderedMessage
	n    int
}

func (s *recordingSender) Channel() Channel { return s.ch }
func (s *recordingSender) Send(_ context.Context, msg RenderedMessage) error {
	s.last = msg
	s.n++
	return nil
}

func newTestDispatcher() (*Dispatcher, map[Channel]*recordingSender) {
	senders := map[Channel]*recordingSender{
		ChannelInApp: {ch: ChannelInApp},
		ChannelSMS:   {ch: ChannelSMS},
		ChannelEmail: {ch: ChannelEmail},
	}
	d := NewDispatcher(fakeTranslator{}, "tr",
		senders[ChannelInApp], senders[ChannelSMS], senders[ChannelEmail])
	return d, senders
}

func TestDispatcher_RoutesByChannel(t *testing.T) {
	d, senders := newTestDispatcher()

	cases := []struct {
		name string
		cmd  SendCommand
		want Channel
	}{
		{"inapp", SendCommand{Channel: ChannelInApp, UserID: "u1", TitleKey: "t", ContentKey: "c"}, ChannelInApp},
		{"sms", SendCommand{Channel: ChannelSMS, Phone: "+905551112233", ContentKey: "c"}, ChannelSMS},
		{"email", SendCommand{Channel: ChannelEmail, Email: "a@b.com", TitleKey: "t", ContentKey: "c"}, ChannelEmail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := d.Send(context.Background(), tc.cmd); err != nil {
				t.Fatalf("Send: %v", err)
			}
			if senders[tc.want].n != 1 {
				t.Fatalf("kanal %q çağrılmadı", tc.want)
			}
		})
	}
}

func TestDispatcher_ValidationPerType(t *testing.T) {
	d, _ := newTestDispatcher()

	cases := []struct {
		name string
		cmd  SendCommand
		want error
	}{
		{"inapp alıcısız", SendCommand{Channel: ChannelInApp, TitleKey: "t", ContentKey: "c"}, ErrRecipientRequired},
		{"sms numarasız", SendCommand{Channel: ChannelSMS, ContentKey: "c"}, ErrPhoneRequired},
		{"email adressiz", SendCommand{Channel: ChannelEmail, TitleKey: "t", ContentKey: "c"}, ErrEmailRequired},
		{"içeriksiz", SendCommand{Channel: ChannelEmail, Email: "a@b.com", TitleKey: "t"}, ErrContentRequired},
		{"bilinmeyen tür", SendCommand{Channel: "push", ContentKey: "c"}, ErrUnsupportedChannel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := d.Send(context.Background(), tc.cmd)
			if !errors.Is(err, tc.want) {
				t.Fatalf("hata = %v, beklenen %v", err, tc.want)
			}
			var de *shared.DomainError
			if !errors.As(err, &de) || de.Kind != shared.KindValidation {
				t.Fatalf("doğrulama hatası bekleniyordu: %v", err)
			}
		})
	}
}

func TestDispatcher_RendersContent(t *testing.T) {
	d, senders := newTestDispatcher()
	err := d.Send(context.Background(), SendCommand{
		Channel: ChannelSMS, Phone: "+905551112233", ContentKey: "sms.body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := senders[ChannelSMS].last.Content; got != "sms.body" {
		t.Fatalf("content = %q", got)
	}
}

func TestDispatcher_RendersHTMLEmail(t *testing.T) {
	d, senders := newTestDispatcher()
	err := d.Send(context.Background(), SendCommand{
		Channel:          ChannelEmail,
		Email:            "a@b.com",
		TitleKey:         "t",
		ContentKey:       "text.body",
		HTMLContentKey:   "html.body",
		TitleFallback:    "subj",
		BodyFallback:     "plain",
		HTMLBodyFallback: "<p>plain</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := senders[ChannelEmail].last
	if msg.Content != "text.body" {
		t.Fatalf("text = %q", msg.Content)
	}
	if msg.HTMLContent != "html.body" {
		t.Fatalf("html = %q", msg.HTMLContent)
	}
}
