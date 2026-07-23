package email

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	dshared "github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
	infranotif "github.com/zatrano/gocore/internal/infrastructure/notification"
	infraoutbox "github.com/zatrano/gocore/internal/infrastructure/outbox"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/pagination"
)

type recordingMailer struct {
	last appshared.Email
}

func (m *recordingMailer) Send(_ context.Context, e appshared.Email) error {
	m.last = e
	return nil
}

type captureEnqueuer struct {
	jobs []appoutbox.Job
}

func (e *captureEnqueuer) Enqueue(_ context.Context, job appoutbox.Job) error {
	e.jobs = append(e.jobs, job)
	return nil
}

type trAdapter struct{ tr *i18n.Translator }

func (a trAdapter) T(locale, key, fallback string, args ...any) string {
	return a.tr.T(i18n.Locale(locale), key, fallback, args...)
}

type stubUserRepo struct {
	u *user.User
}

func (r *stubUserRepo) Save(context.Context, *user.User) error                { return nil }
func (r *stubUserRepo) Update(context.Context, *user.User) error              { return nil }
func (r *stubUserRepo) FindByID(context.Context, user.ID) (*user.User, error) { return r.u, nil }
func (r *stubUserRepo) FindByIDIncludeDeleted(context.Context, user.ID) (*user.User, error) {
	return r.u, nil
}
func (r *stubUserRepo) FindByEmail(context.Context, user.Email) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (r *stubUserRepo) ExistsByEmail(context.Context, user.Email) (bool, error)    { return false, nil }
func (r *stubUserRepo) FindByIDs(context.Context, []user.ID) ([]*user.User, error) { return nil, nil }
func (r *stubUserRepo) List(context.Context, user.ListFilter, pagination.Request) (pagination.Page[*user.User], error) {
	return pagination.Page[*user.User]{}, nil
}
func (r *stubUserRepo) Delete(context.Context, user.ID) error     { return nil }
func (r *stubUserRepo) Restore(context.Context, user.ID) error    { return nil }
func (r *stubUserRepo) HardDelete(context.Context, user.ID) error { return nil }
func (r *stubUserRepo) CountActiveByRole(context.Context, user.Role) (int64, error) {
	return 0, nil
}

func drainDispatch(t *testing.T, enq *captureEnqueuer, d *appnotif.Dispatcher) {
	t.Helper()
	h := infraoutbox.DispatchHandler(d)
	for _, job := range enq.jobs {
		if err := h(context.Background(), job); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUserNotifier_OnRegistered_UsesProfileLocale(t *testing.T) {
	tr, err := i18n.NewFromEmbedded("tr", []i18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	mailer := &recordingMailer{}
	enq := &captureEnqueuer{}
	n := NewUserNotifier(NewOutboxDispatcher(enq), &stubUserRepo{})
	d := appnotif.NewDispatcher(trAdapter{tr: tr}, "tr", infranotif.NewEmailChannel(mailer))

	ev := user.RegisteredEvent{
		BaseEvent:       dshared.NewBaseEvent(user.EventRegistered, "user-id"),
		Email:           "alice@example.com",
		Name:            "Alice",
		PreferredLocale: "en",
	}
	payload, _ := json.Marshal(ev)
	if err := n.OnRegisteredPayload(context.Background(), appoutbox.DomainEventPayload{
		EventID: ev.EventID(), EventName: user.EventRegistered, AggregateID: "user-id", Data: payload,
	}); err != nil {
		t.Fatal(err)
	}
	drainDispatch(t, enq, d)
	if mailer.last.Subject != "Welcome, Alice!" {
		t.Errorf("subject = %q", mailer.last.Subject)
	}
	if mailer.last.HTMLBody == "" {
		t.Error("expected HTML body")
	}
}

func TestUserNotifier_OnActivated_UsesStoredLocale(t *testing.T) {
	tr, err := i18n.NewFromEmbedded("tr", []i18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	emailVO, _ := user.NewEmail("bob@example.com")
	hashed, _ := user.NewHashedPassword("$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$hash")
	loc, _ := user.ParsePreferredLocale("tr")
	now := time.Now().UTC()
	u := user.Hydrate(user.NewID(), emailVO, user.Phone{}, "Bob", hashed, user.RoleUser, true, true, false, "", loc, now, now, nil)

	mailer := &recordingMailer{}
	enq := &captureEnqueuer{}
	n := NewUserNotifier(NewOutboxDispatcher(enq), &stubUserRepo{u: u})
	d := appnotif.NewDispatcher(trAdapter{tr: tr}, "tr", infranotif.NewEmailChannel(mailer))

	ev := user.ActivatedEvent{BaseEvent: dshared.NewBaseEvent(user.EventActivated, u.ID().String())}
	payload, _ := json.Marshal(ev)
	if err := n.OnActivatedPayload(context.Background(), appoutbox.DomainEventPayload{
		EventID: ev.EventID(), EventName: user.EventActivated, AggregateID: u.ID().String(), Data: payload,
	}); err != nil {
		t.Fatal(err)
	}
	drainDispatch(t, enq, d)
	if mailer.last.Subject != "Hesabınız aktifleştirildi" {
		t.Errorf("subject = %q", mailer.last.Subject)
	}
}
