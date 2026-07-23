package security

import (
	"context"
	"sync"
	"time"
)

// MemoryLoginGuard, auth.LoginGuard portunun bellek içi implementasyonudur.
// Tek node için uygundur; çok node'lu (stateless) dağıtımda Redis tabanlı bir
// implementasyonla değiştirilmelidir (port sayesinde kolay). Eşik aşılınca
// anahtar belirli süre kilitlenir.
type MemoryLoginGuard struct {
	mu          sync.Mutex
	attempts    map[string]*attemptState
	maxAttempts int
	lockout     time.Duration
	now         func() time.Time
}

type attemptState struct {
	count      int
	lockedTill time.Time
}

// NewMemoryLoginGuard, guard'ı maksimum deneme ve kilit süresiyle kurar.
func NewMemoryLoginGuard(maxAttempts int, lockout time.Duration) *MemoryLoginGuard {
	return &MemoryLoginGuard{
		attempts:    make(map[string]*attemptState),
		maxAttempts: maxAttempts,
		lockout:     lockout,
		now:         time.Now,
	}
}

// Allowed, anahtarın kilitli olup olmadığını kontrol eder.
func (g *MemoryLoginGuard) Allowed(_ context.Context, key string) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	st, ok := g.attempts[key]
	if !ok {
		return true, nil
	}
	// Kilit süresi dolmuşsa sıfırla.
	if !st.lockedTill.IsZero() && g.now().After(st.lockedTill) {
		delete(g.attempts, key)
		return true, nil
	}
	return st.lockedTill.IsZero(), nil
}

// RecordFailure, başarısız denemeyi sayar; eşik aşılırsa kilitler.
func (g *MemoryLoginGuard) RecordFailure(_ context.Context, key string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	st, ok := g.attempts[key]
	if !ok {
		st = &attemptState{}
		g.attempts[key] = st
	}
	st.count++
	if st.count >= g.maxAttempts {
		st.lockedTill = g.now().Add(g.lockout)
	}
	return nil
}

// Reset, başarılı girişte sayaçları temizler.
func (g *MemoryLoginGuard) Reset(_ context.Context, key string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.attempts, key)
	return nil
}

// Cleanup, süresi dolmuş kayıtları temizler (arka plan job'undan periyodik
// olarak çağrılabilir; bellek sızıntısını önler).
func (g *MemoryLoginGuard) Cleanup() {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := g.now()
	for k, st := range g.attempts {
		if !st.lockedTill.IsZero() && now.After(st.lockedTill) {
			delete(g.attempts, k)
		}
	}
}
