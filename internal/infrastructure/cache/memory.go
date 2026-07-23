// Package cache, appshared.Cache portunun implementasyonlarını içerir.
package cache

import (
	"context"
	"sync"
	"time"
)

// Memory, TTL destekli, thread-safe bellek içi cache'tir. Tek node için uygundur;
// çok node'lu dağıtımda Redis implementasyonuyla değiştirilir (port aynı kalır).
type Memory struct {
	mu    sync.RWMutex
	items map[string]item
	now   func() time.Time
}

type item struct {
	value     []byte
	expiresAt time.Time
}

// NewMemory, cache'i kurar.
func NewMemory() *Memory {
	return &Memory{items: make(map[string]item), now: time.Now}
}

// Get, anahtarın değerini döner. Süresi dolmuşsa yok sayılır.
func (m *Memory) Get(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	it, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !it.expiresAt.IsZero() && m.now().After(it.expiresAt) {
		m.mu.Lock()
		delete(m.items, key)
		m.mu.Unlock()
		return nil, false, nil
	}
	// Zero-copy istenmiyorsa çağıran mutasyon yapabilir; güvenlik için kopyalarız.
	out := make([]byte, len(it.value))
	copy(out, it.value)
	return out, true, nil
}

// Take, anahtarın değerini aynı kilit altında okuyup siler.
func (m *Memory) Take(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	it, ok := m.items[key]
	if !ok {
		return nil, false, nil
	}
	delete(m.items, key)
	if !it.expiresAt.IsZero() && m.now().After(it.expiresAt) {
		return nil, false, nil
	}
	out := make([]byte, len(it.value))
	copy(out, it.value)
	return out, true, nil
}

// Set, değeri TTL ile yazar (ttl<=0 ise süresiz).
func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = m.now().Add(ttl)
	}
	stored := make([]byte, len(value))
	copy(stored, value)

	m.mu.Lock()
	m.items[key] = item{value: stored, expiresAt: exp}
	m.mu.Unlock()
	return nil
}

// Delete, anahtarı siler.
func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

// Cleanup, süresi dolmuş girdileri temizler (arka plan job'undan çağrılabilir).
func (m *Memory) Cleanup() {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, it := range m.items {
		if !it.expiresAt.IsZero() && now.After(it.expiresAt) {
			delete(m.items, k)
		}
	}
}
