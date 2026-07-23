package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/zatrano/gocore/internal/domain/user"
)

// syncRunner, görevi anında çalıştırır (test için).
type syncRunner struct{}

func (syncRunner) Go(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func newTestManualSender() (*ManualSender, map[Channel]*recordingSender) {
	d, senders := newTestDispatcher()
	return NewManualSender(ManualSenderDeps{Dispatcher: d, Runner: syncRunner{}}), senders
}

func TestManualSender_SendOne(t *testing.T) {
	s, senders := newTestManualSender()
	err := s.SendOne(context.Background(), ChannelEmail,
		Recipient{Email: "a@b.com"},
		MessageContent{Title: "Merhaba", Body: "Deneme"})
	if err != nil {
		t.Fatalf("SendOne: %v", err)
	}
	if senders[ChannelEmail].n != 1 {
		t.Fatal("email kanalı çağrılmadı")
	}
	if senders[ChannelEmail].last.Content != "Deneme" || senders[ChannelEmail].last.Title != "Merhaba" {
		t.Fatalf("içerik yanlış: %+v", senders[ChannelEmail].last)
	}
}

func TestManualSender_SendOne_ValidationError(t *testing.T) {
	s, _ := newTestManualSender()
	err := s.SendOne(context.Background(), ChannelSMS,
		Recipient{}, // telefon yok
		MessageContent{Body: "x"})
	if !errors.Is(err, ErrPhoneRequired) {
		t.Fatalf("ErrPhoneRequired bekleniyordu, alınan: %v", err)
	}
}

func TestManualSender_SendBulk_Summary(t *testing.T) {
	s, senders := newTestManualSender()
	res, err := s.SendBulk(context.Background(), BulkSendCommand{
		Channel: ChannelSMS,
		Content: MessageContent{Body: "Kampanya"},
		Recipients: []Recipient{
			{Phone: "+905551112233"},
			{Phone: ""}, // geçersiz
			{Phone: "+905553334455"},
		},
	})
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if res.Total != 3 || res.Accepted != 2 || len(res.Invalid) != 1 {
		t.Fatalf("özet yanlış: %+v", res)
	}
	if res.Invalid[0].Index != 1 || res.Invalid[0].Reason != "notification.phone_required" {
		t.Fatalf("geçersiz kayıt yanlış: %+v", res.Invalid[0])
	}
	if senders[ChannelSMS].n != 2 {
		t.Fatalf("2 sms gönderilmeliydi, gönderilen %d", senders[ChannelSMS].n)
	}
}

func TestManualSender_SendBulk_EmptyBody(t *testing.T) {
	s, _ := newTestManualSender()
	_, err := s.SendBulk(context.Background(), BulkSendCommand{
		Channel:    ChannelEmail,
		Content:    MessageContent{Title: "t"},
		Recipients: []Recipient{{Email: "a@b.com"}},
	})
	if !errors.Is(err, ErrContentRequired) {
		t.Fatalf("ErrContentRequired bekleniyordu, alınan: %v", err)
	}
}

func TestManualSender_SendBulk_UnsupportedChannel(t *testing.T) {
	s, _ := newTestManualSender()
	_, err := s.SendBulk(context.Background(), BulkSendCommand{
		Channel:    "push",
		Content:    MessageContent{Body: "x"},
		Recipients: []Recipient{{UserID: "u1"}},
	})
	if !errors.Is(err, ErrUnsupportedChannel) {
		t.Fatalf("ErrUnsupportedChannel bekleniyordu, alınan: %v", err)
	}
}

func TestManualSender_SendBulk_LocaleFilter(t *testing.T) {
	s, senders := newTestManualSender()
	res, err := s.SendBulk(context.Background(), BulkSendCommand{
		Channel: ChannelSMS,
		Content: MessageContent{Body: "x"},
		Locale:  "en",
		Recipients: []Recipient{
			{Phone: "+905551112233"},               // dili yok → hedef dil uygulanır
			{Phone: "+905553334455", Locale: "tr"}, // uyuşmaz → invalid
			{Phone: "+905554445566", Locale: "en"}, // eşleşir
		},
	})
	if err != nil {
		t.Fatalf("SendBulk: %v", err)
	}
	if res.Total != 3 || res.Accepted != 2 || len(res.Invalid) != 1 {
		t.Fatalf("özet yanlış: %+v", res)
	}
	if res.Invalid[0].Index != 1 || res.Invalid[0].Reason != "locale_mismatch" {
		t.Fatalf("geçersiz kayıt yanlış: %+v", res.Invalid[0])
	}
	if senders[ChannelSMS].n != 2 {
		t.Fatalf("2 sms gönderilmeliydi, gönderilen %d", senders[ChannelSMS].n)
	}
	if senders[ChannelSMS].last.Locale != "en" {
		t.Fatalf("locale = %q, beklenen en", senders[ChannelSMS].last.Locale)
	}
}

type stubUserDirectory struct {
	pages [][]UserContact
	err   error
}

func (s stubUserDirectory) ListActiveContacts(_ context.Context, pageNum int, _ int) ([]UserContact, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	idx := pageNum - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s.pages) {
		return nil, false, nil
	}
	hasMore := idx+1 < len(s.pages)
	return s.pages[idx], hasMore, nil
}

func TestManualSender_SendToAllUsers_InApp(t *testing.T) {
	d, senders := newTestDispatcher()
	id1 := user.NewID()
	id2 := user.NewID()
	id3 := user.NewID()
	dir := stubUserDirectory{pages: [][]UserContact{
		{
			{ID: id1.String(), Email: "a@b.com", Locale: "tr"},
			{ID: id2.String(), Email: "c@d.com", Locale: "en"},
			{ID: id3.String(), Email: "e@f.com", Locale: "en"},
		},
	}}
	repo := &stubUserRepo{byID: map[string]*user.User{
		id1.String(): activeTestUserWithLocale(id1, "a@b.com", "tr"),
		id2.String(): activeTestUserWithLocale(id2, "c@d.com", "en"),
		id3.String(): activeTestUserWithLocale(id3, "e@f.com", "en"),
	}}
	s := NewManualSender(ManualSenderDeps{Dispatcher: d, Users: dir, Resolver: UserRepoResolver{Users: repo}})
	res, err := s.SendToAllUsers(context.Background(), ChannelInApp,
		MessageContent{Title: "Hello", Body: "Notice"}, "en")
	if err != nil {
		t.Fatalf("SendToAllUsers: %v", err)
	}
	if res.Total != 2 || res.Accepted != 2 {
		t.Fatalf("özet yanlış: %+v", res)
	}
	if senders[ChannelInApp].n != 2 {
		t.Fatalf("2 inapp gönderilmeliydi, gönderilen %d", senders[ChannelInApp].n)
	}
	if senders[ChannelInApp].last.Locale != "en" {
		t.Fatalf("dil = %q, beklenen en", senders[ChannelInApp].last.Locale)
	}
}

func TestManualSender_SendToAllUsers_SMS(t *testing.T) {
	d, senders := newTestDispatcher()
	dir := stubUserDirectory{pages: [][]UserContact{
		{
			{ID: "u1", Email: "a@b.com", Phone: "+905551112233", Locale: "tr"},
			{ID: "u2", Email: "c@d.com", Locale: "en"}, // dil uyuşmaz → listede yok
			{ID: "u3", Email: "e@f.com", Locale: "tr"}, // telefon yok → invalid
		},
	}}
	s := NewManualSender(ManualSenderDeps{Dispatcher: d, Runner: syncRunner{}, Users: dir})
	res, err := s.SendToAllUsers(context.Background(), ChannelSMS,
		MessageContent{Body: "Kampanya"}, "tr")
	if err != nil {
		t.Fatalf("SendToAllUsers: %v", err)
	}
	if res.Total != 2 || res.Accepted != 1 || len(res.Invalid) != 1 {
		t.Fatalf("özet yanlış: %+v", res)
	}
	if senders[ChannelSMS].n != 1 {
		t.Fatalf("1 sms gönderilmeliydi, gönderilen %d", senders[ChannelSMS].n)
	}
}

func TestManualSender_SendToAllUsers_DirectoryRequired(t *testing.T) {
	s, _ := newTestManualSender()
	_, err := s.SendToAllUsers(context.Background(), ChannelEmail,
		MessageContent{Title: "t", Body: "b"}, "tr")
	if !errors.Is(err, ErrUserDirectoryRequired) {
		t.Fatalf("ErrUserDirectoryRequired bekleniyordu, alınan: %v", err)
	}
}
