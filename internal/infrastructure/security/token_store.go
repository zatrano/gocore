package security

import (
	"context"
	"sync"
	"time"
)

// MemoryTokenStore, auth.TokenStore portunun bellek içi implementasyonudur.
// Refresh token rotation, yeniden kullanım (reuse) tespiti ve access token
// iptalini yönetir. Tek node için uygundur; çok node'lu dağıtımda Redis ile
// değiştirilebilir (port sayesinde kolay).
type MemoryTokenStore struct {
	mu sync.Mutex
	// activeRefresh[userID][tokenID] = expiry. Yalnızca tüketilmemiş token'lar.
	activeRefresh map[string]map[string]time.Time
	// consumedRefresh[userID][tokenID] = expiry. Rotation/logout sonrası reuse tespiti.
	consumedRefresh map[string]map[string]time.Time
	// revokedAccess[tokenID] = expiry. Logout ile iptal edilen access token'lar.
	revokedAccess map[string]time.Time
	// userRevokedAt[userID] = zaman. Bu andan önce üretilen tüm token'lar geçersiz.
	userRevokedAt map[string]time.Time
	now           func() time.Time
}

// NewMemoryTokenStore, store'u kurar.
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		activeRefresh:   make(map[string]map[string]time.Time),
		consumedRefresh: make(map[string]map[string]time.Time),
		revokedAccess:   make(map[string]time.Time),
		userRevokedAt:   make(map[string]time.Time),
		now:             time.Now,
	}
}

// ActivateRefresh, yeni refresh token'ı aktif kaydeder.
func (s *MemoryTokenStore) ActivateRefresh(_ context.Context, userID, tokenID string, exp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.activeRefresh[userID]
	if !ok {
		m = make(map[string]time.Time)
		s.activeRefresh[userID] = m
	}
	m[tokenID] = exp
	return nil
}

// IsRefreshActive, refresh token hâlâ geçerli mi?
func (s *MemoryTokenStore) IsRefreshActive(_ context.Context, userID, tokenID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.activeRefresh[userID]
	if !ok {
		return false, nil
	}
	exp, ok := m[tokenID]
	if !ok {
		return false, nil
	}
	if s.now().After(exp) {
		delete(m, tokenID)
		return false, nil
	}
	return true, nil
}

// WasRefreshConsumed, refresh token daha önce tüketildi mi?
func (s *MemoryTokenStore) WasRefreshConsumed(_ context.Context, userID, tokenID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.consumedRefresh[userID]
	if !ok {
		return false, nil
	}
	exp, ok := m[tokenID]
	if !ok {
		return false, nil
	}
	if s.now().After(exp) {
		delete(m, tokenID)
		return false, nil
	}
	return true, nil
}

// ConsumeRefresh, rotation'da eski refresh token'ı tüketir.
func (s *MemoryTokenStore) ConsumeRefresh(_ context.Context, userID, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp := s.now().Add(24 * time.Hour)
	if m, ok := s.activeRefresh[userID]; ok {
		if e, ok := m[tokenID]; ok {
			exp = e
		}
		delete(m, tokenID)
	}
	cm, ok := s.consumedRefresh[userID]
	if !ok {
		cm = make(map[string]time.Time)
		s.consumedRefresh[userID] = cm
	}
	cm[tokenID] = exp
	return nil
}

// RevokeAccess, access token'ı süresi dolana dek iptal eder.
func (s *MemoryTokenStore) RevokeAccess(_ context.Context, tokenID string, exp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedAccess[tokenID] = exp
	return nil
}

// IsAccessRevoked, access token iptal edilmiş mi?
func (s *MemoryTokenStore) IsAccessRevoked(_ context.Context, tokenID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.revokedAccess[tokenID]
	return ok, nil
}

// RevokeAllForUser, kullanıcının tüm oturumlarını iptal eder.
func (s *MemoryTokenStore) RevokeAllForUser(_ context.Context, userID string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeRefresh, userID)
	delete(s.consumedRefresh, userID)
	s.userRevokedAt[userID] = at
	return nil
}

// UserRevokedAt, kullanıcının toplu iptal zaman damgasını döner.
func (s *MemoryTokenStore) UserRevokedAt(_ context.Context, userID string) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userRevokedAt[userID], nil
}

// Cleanup, süresi dolmuş kayıtları temizler (periyodik job'dan çağrılır).
func (s *MemoryTokenStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for id, exp := range s.revokedAccess {
		if now.After(exp) {
			delete(s.revokedAccess, id)
		}
	}
	for userID, m := range s.activeRefresh {
		for id, exp := range m {
			if now.After(exp) {
				delete(m, id)
			}
		}
		if len(m) == 0 {
			delete(s.activeRefresh, userID)
		}
	}
	for userID, m := range s.consumedRefresh {
		for id, exp := range m {
			if now.After(exp) {
				delete(m, id)
			}
		}
		if len(m) == 0 {
			delete(s.consumedRefresh, userID)
		}
	}
}
