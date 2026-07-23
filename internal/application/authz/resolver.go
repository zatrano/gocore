package authz

import (
	"context"
	"sort"
	"sync"
	"time"

	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
	"github.com/zatrano/gocore/pkg/rbac"
)

// Resolver, rbac.Checker arayüzünü veritabanı destekli olarak gerçekler.
type Resolver struct {
	repo domainrbac.Repository
	ttl  time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	perms     map[string]struct{}
	expiresAt time.Time
}

// NewResolver, verilen repository ve önbellek TTL'i ile bir Resolver kurar.
func NewResolver(repo domainrbac.Repository, ttl time.Duration) *Resolver {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Resolver{repo: repo, ttl: ttl, cache: make(map[string]cacheEntry)}
}

var _ rbac.Checker = (*Resolver)(nil)

func (r *Resolver) load(ctx context.Context, role string) (map[string]struct{}, error) {
	r.mu.RLock()
	entry, ok := r.cache[role]
	r.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.perms, nil
	}

	rn, err := domainrbac.ParseRoleName(role)
	if err != nil {
		return map[string]struct{}{}, nil //nolint:nilerr // geçersiz rol adı: boş izin kümesi
	}
	found, err := r.repo.FindByName(ctx, rn)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(found.Permissions()))
	for _, p := range found.PermissionNames() {
		set[p] = struct{}{}
	}

	r.mu.Lock()
	r.cache[role] = cacheEntry{perms: set, expiresAt: time.Now().Add(r.ttl)}
	r.mu.Unlock()
	return set, nil
}

// Allows, rolün belirtilen izne sahip olup olmadığını döner.
func (r *Resolver) Allows(ctx context.Context, role string, perm rbac.Permission) (bool, error) {
	set, err := r.load(ctx, role)
	if err != nil {
		return false, err
	}
	_, ok := set[string(perm)]
	return ok, nil
}

// PermissionsFor, rolün sahip olduğu izinleri sıralı olarak döner.
func (r *Resolver) PermissionsFor(ctx context.Context, role string) ([]string, error) {
	set, err := r.load(ctx, role)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// Invalidate, tüm önbelleği temizler.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	r.cache = make(map[string]cacheEntry)
	r.mu.Unlock()
}
