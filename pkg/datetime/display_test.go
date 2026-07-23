package datetime

import (
	"testing"
	"time"
)

func TestFormatDateTime_istanbulOffset(t *testing.T) {
	utc := time.Date(2026, 7, 10, 11, 39, 21, 0, time.UTC)
	got := FormatDateTime(utc)
	want := "10.07.2026 14:39:21"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatDateTime_zero(t *testing.T) {
	if FormatDateTime(time.Time{}) != "" {
		t.Fatal("expected empty for zero time")
	}
}

func TestJSONTime_MarshalJSON(t *testing.T) {
	utc := time.Date(2026, 7, 10, 11, 39, 21, 0, time.UTC)
	b, err := FromTime(utc).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2026-07-10T14:39:21+03:00"` {
		t.Fatalf("unexpected %s", b)
	}
}
