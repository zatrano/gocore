package pagination_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/zatrano/gocore/pkg/pagination"
)

func TestEncodeDecodeCursor_roundtrip(t *testing.T) {
	at := time.Date(2024, 6, 15, 12, 30, 0, 123456789, time.UTC)
	id := uuid.NewString()

	encoded := pagination.EncodeCursor(at, id)
	if encoded == "" {
		t.Fatal("expected non-empty cursor")
	}

	gotAt, gotID, err := pagination.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotAt.Equal(at) {
		t.Fatalf("created_at: got %v want %v", gotAt, at)
	}
	if gotID != id {
		t.Fatalf("id: got %q want %q", gotID, id)
	}
}

func TestDecodeCursor_invalid(t *testing.T) {
	cases := []string{"", "not-base64", "!!!", "dGVzdA"} // last is valid b64 but bad JSON
	for _, s := range cases {
		if _, _, err := pagination.DecodeCursor(s); err == nil {
			t.Fatalf("expected error for %q", s)
		}
	}
}

func TestEncodeNextCursor(t *testing.T) {
	at := time.Now().UTC()
	items := []struct {
		at time.Time
		id string
	}{
		{at, uuid.NewString()},
		{at.Add(time.Second), uuid.NewString()},
	}
	key := func(it struct {
		at time.Time
		id string
	}) (time.Time, string) {
		return it.at, it.id
	}

	if got := pagination.EncodeNextCursor(items, 3, key); got != "" {
		t.Fatalf("expected empty when len < limit, got %q", got)
	}

	next := pagination.EncodeNextCursor(items, 2, key)
	if next == "" {
		t.Fatal("expected next cursor")
	}
	_, id, err := pagination.DecodeCursor(next)
	if err != nil {
		t.Fatalf("decode next: %v", err)
	}
	if id != items[1].id {
		t.Fatalf("next id: got %q want %q", id, items[1].id)
	}
}
