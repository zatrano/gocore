package security

import (
	"testing"
	"time"
)

func TestMemoryIPRateLimiterAllowsUntilMax(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	lim := NewMemoryIPRateLimiter(3, time.Minute)
	lim.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !lim.Allow("1.1.1.1") {
			t.Fatalf("attempt %d should allow", i+1)
		}
	}
	if lim.Allow("1.1.1.1") {
		t.Fatal("4th attempt should deny")
	}
	if !lim.Allow("2.2.2.2") {
		t.Fatal("other key should allow")
	}

	now = now.Add(time.Minute + time.Second)
	if !lim.Allow("1.1.1.1") {
		t.Fatal("after window should allow again")
	}
}
