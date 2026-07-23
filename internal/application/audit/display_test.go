package audit

import "testing"

func TestFormatChangeSummary_email(t *testing.T) {
	got := formatChangeSummary("user.email_changed", map[string]any{
		"old_email": "a@x.com", "new_email": "b@x.com",
	})
	if got != "a@x.com → b@x.com" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatChangeSummary_emptyMeta(t *testing.T) {
	if formatChangeSummary("user.deleted", nil) != "" {
		t.Fatal("expected empty")
	}
}
