package contact_test

import (
	"testing"

	"github.com/zatrano/gocore/internal/domain/contact"
)

func TestSubmit_Valid(t *testing.T) {
	m, err := contact.Submit("Ada Lovelace", "ada@example.com", "Merhaba, bilgi almak istiyorum.", "tr", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if m.Status() != contact.StatusReceived {
		t.Fatalf("status %s", m.Status())
	}
	events := m.PullEvents()
	if len(events) != 1 || events[0].EventName() != contact.EventSubmitted {
		t.Fatalf("events %#v", events)
	}
	if events[0].EventID() == "" {
		t.Fatal("empty event id")
	}
}

func TestSubmit_Invalid(t *testing.T) {
	if _, err := contact.Submit("A", "bad", "x", "tr", "", ""); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestMarkRead(t *testing.T) {
	m, err := contact.Submit("Ada Lovelace", "ada@example.com", "Merhaba, bilgi almak istiyorum.", "tr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if m.IsRead() || m.ReadAt() != nil {
		t.Fatal("expected unread")
	}
	m.MarkRead()
	if !m.IsRead() || m.ReadAt() == nil {
		t.Fatal("expected read")
	}
	first := *m.ReadAt()
	m.MarkRead()
	if !first.Equal(*m.ReadAt()) {
		t.Fatal("MarkRead should be idempotent")
	}
}
