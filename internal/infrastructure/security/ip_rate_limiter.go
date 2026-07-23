package security

import (
	"sync"
	"time"
)

// MemoryIPRateLimiter, IP (veya herhangi bir anahtar) başına sabit pencereli
// istek kotası uygular. GoUI WebSocket olayları Fiber limiter dışında kaldığı
// için application seviyesinde kullanılır.
type MemoryIPRateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	max     int
	window  time.Duration
	now     func() time.Time
	cleanup time.Time
}

// NewMemoryIPRateLimiter, pencere ve üst sınır ile limiter kurar.
func NewMemoryIPRateLimiter(max int, window time.Duration) *MemoryIPRateLimiter {
	if max < 1 {
		max = 100
	}
	if window <= 0 {
		window = time.Minute
	}
	return &MemoryIPRateLimiter{
		hits:   make(map[string][]time.Time),
		max:    max,
		window: window,
		now:    time.Now,
	}
}

// Allow, anahtarın pencere içindeki kotasını tüketir. Kota aşılırsa false döner.
func (l *MemoryIPRateLimiter) Allow(key string) bool {
	if l == nil || key == "" {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-l.window)
	if now.After(l.cleanup) {
		l.pruneLocked(cutoff)
		l.cleanup = now.Add(l.window)
	}

	existing := l.hits[key]
	recent := make([]time.Time, 0, len(existing)+1)
	for _, ts := range existing {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}
	if len(recent) >= l.max {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}

func (l *MemoryIPRateLimiter) pruneLocked(cutoff time.Time) {
	for key, stamps := range l.hits {
		kept := stamps[:0]
		for _, ts := range stamps {
			if ts.After(cutoff) {
				kept = append(kept, ts)
			}
		}
		if len(kept) == 0 {
			delete(l.hits, key)
			continue
		}
		l.hits[key] = kept
	}
}
