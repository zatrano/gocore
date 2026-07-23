package contact_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	appcontact "github.com/zatrano/gocore/internal/application/contact"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	domaincontact "github.com/zatrano/gocore/internal/domain/contact"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/pagination"
)

type memRepo struct {
	byID map[string]*domaincontact.Message
}

func newMemRepo() *memRepo {
	return &memRepo{byID: map[string]*domaincontact.Message{}}
}

func (r *memRepo) Save(_ context.Context, m *domaincontact.Message) error {
	r.byID[m.ID().String()] = m
	return nil
}

func (r *memRepo) FindByID(_ context.Context, id domaincontact.ID) (*domaincontact.Message, error) {
	m, ok := r.byID[id.String()]
	if !ok {
		return nil, domaincontact.ErrNotFound
	}
	return m, nil
}

func (r *memRepo) List(_ context.Context, page pagination.Request, unreadOnly bool) (pagination.Page[*domaincontact.Message], error) {
	items := make([]*domaincontact.Message, 0, len(r.byID))
	for _, m := range r.byID {
		if unreadOnly && m.IsRead() {
			continue
		}
		items = append(items, m)
	}
	return pagination.NewPage(items, page.Page, page.Limit, int64(len(items))), nil
}

func (r *memRepo) MarkRead(_ context.Context, id domaincontact.ID) error {
	m, ok := r.byID[id.String()]
	if !ok {
		return domaincontact.ErrNotFound
	}
	m.MarkRead()
	return nil
}

type memOutbox struct {
	jobs []appoutbox.Job
	err  error
}

func (o *memOutbox) Enqueue(_ context.Context, job appoutbox.Job) error {
	if o.err != nil {
		return o.err
	}
	o.jobs = append(o.jobs, job)
	return nil
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, ...shared.DomainEvent) error { return nil }

type passthroughTx struct{}

func (passthroughTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestSubmitHandler_enqueuesEscapedHTMLEmail(t *testing.T) {
	repo := newMemRepo()
	outbox := &memOutbox{}
	h := appcontact.NewSubmitHandler(repo, outbox, noopPublisher{}, passthroughTx{}, "admin@example.com")

	view, err := h.Handle(context.Background(), appcontact.SubmitCommand{
		Name:    `<script>alert(1)</script>`,
		Email:   "user@example.com",
		Message: "Merhaba\n<script>x</script>",
		Locale:  "tr",
		IP:      "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if view.ID == "" {
		t.Fatal("expected id")
	}
	if len(outbox.jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(outbox.jobs))
	}
	var payload appoutbox.EmailPayload
	if err := json.Unmarshal(outbox.jobs[0].Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if strings.Contains(payload.HTMLBody, "<script>") {
		t.Fatalf("HTML body must escape scripts: %s", payload.HTMLBody)
	}
	if !strings.Contains(payload.HTMLBody, "&lt;script&gt;") {
		t.Fatalf("expected escaped script tags: %s", payload.HTMLBody)
	}
	if !strings.Contains(payload.HTMLBody, "<br>") {
		t.Fatalf("expected newline to br conversion: %s", payload.HTMLBody)
	}
}

func TestSubmitHandler_skipsEmailWhenNoRecipient(t *testing.T) {
	repo := newMemRepo()
	outbox := &memOutbox{}
	h := appcontact.NewSubmitHandler(repo, outbox, noopPublisher{}, passthroughTx{}, "")
	if _, err := h.Handle(context.Background(), appcontact.SubmitCommand{
		Name: "Ali Veli", Email: "a@b.co", Message: "Merhaba dünya",
	}); err != nil {
		t.Fatal(err)
	}
	if len(outbox.jobs) != 0 {
		t.Fatalf("expected no email job, got %d", len(outbox.jobs))
	}
	if len(repo.byID) != 1 {
		t.Fatal("expected message saved")
	}
}

func TestSubmitHandler_rollsBackOnOutboxError(t *testing.T) {
	repo := newMemRepo()
	outbox := &memOutbox{err: errors.New("enqueue failed")}
	tx := &recordingTx{}
	h := appcontact.NewSubmitHandler(repo, outbox, noopPublisher{}, tx, "admin@example.com")
	_, err := h.Handle(context.Background(), appcontact.SubmitCommand{
		Name: "Ali Veli", Email: "a@b.co", Message: "Merhaba dünya",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !tx.called {
		t.Fatal("expected WithinTx")
	}
}

type recordingTx struct{ called bool }

func (t *recordingTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	t.called = true
	return fn(ctx)
}

func TestListGetMarkRead(t *testing.T) {
	repo := newMemRepo()
	msg, err := domaincontact.Submit("Ali Veli", "a@b.co", "Merhaba dünya", "tr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = repo.Save(context.Background(), msg)

	listH := appcontact.NewListHandler(repo)
	page, err := listH.Handle(context.Background(), appcontact.ListQuery{Page: 1, Limit: 10, UnreadOnly: true})
	if err != nil || page.Total != 1 {
		t.Fatalf("list unread: %+v err=%v", page, err)
	}

	getH := appcontact.NewGetHandler(repo)
	view, err := getH.Handle(context.Background(), msg.ID().String())
	if err != nil || view.Read {
		t.Fatalf("get: %+v err=%v", view, err)
	}

	markH := appcontact.NewMarkReadHandler(repo)
	view, err = markH.Handle(context.Background(), appcontact.MarkReadCommand{ID: msg.ID().String()})
	if err != nil || !view.Read || view.ReadAt == nil {
		t.Fatalf("mark read: %+v err=%v", view, err)
	}
	firstReadAt := *view.ReadAt
	view2, err := markH.Handle(context.Background(), appcontact.MarkReadCommand{ID: msg.ID().String()})
	if err != nil || view2.ReadAt == nil || !view2.ReadAt.Equal(firstReadAt.Time) {
		t.Fatalf("mark read should be idempotent: %+v", view2)
	}

	page, err = listH.Handle(context.Background(), appcontact.ListQuery{Page: 1, Limit: 10, UnreadOnly: true})
	if err != nil || page.Total != 0 {
		t.Fatalf("expected no unread, got %+v", page)
	}
}

// compile-time stubs for interfaces used above
var (
	_ domaincontact.Repository = (*memRepo)(nil)
	_ appoutbox.Enqueuer       = (*memOutbox)(nil)
	_ appshared.EventPublisher = noopPublisher{}
	_ appshared.TxManager      = passthroughTx{}
	_ appshared.TxManager      = (*recordingTx)(nil)
)
