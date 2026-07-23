package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishToSubscriber(t *testing.T) {
	t.Parallel()
	h := NewHub(func(_ context.Context, userID string) (int64, error) {
		if userID != "u1" {
			t.Fatalf("unexpected user %q", userID)
		}
		return 7, nil
	})
	c := h.Subscribe("u1")
	defer h.Unsubscribe("u1", c)

	h.NotifyInbox("u1")

	select {
	case raw := <-c.Outbound():
		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Type != EventInboxUpdated || ev.UnreadCount != 7 {
			t.Fatalf("got %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestHubDoesNotLeakToOtherUsers(t *testing.T) {
	t.Parallel()
	h := NewHub(nil)
	a := h.Subscribe("a")
	b := h.Subscribe("b")
	defer h.Unsubscribe("a", a)
	defer h.Unsubscribe("b", b)

	h.Publish("a", Event{Type: EventInboxUpdated, UnreadCount: 1})

	select {
	case <-a.Outbound():
	case <-time.After(time.Second):
		t.Fatal("user a should receive")
	}
	select {
	case msg := <-b.Outbound():
		t.Fatalf("user b leaked: %s", msg)
	case <-time.After(50 * time.Millisecond):
	}
}
