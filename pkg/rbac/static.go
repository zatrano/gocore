package rbac

import (
	"context"
	"sort"
)

// StaticChecker, sabit bir rol→izin haritasından Checker arayüzünü gerçekleyen
// bellek-içi uygulamadır. Testler ve veritabanı erişimi olmayan senaryolar
// (ör. birim testleri) için uygundur.
type StaticChecker struct {
	roles map[string]map[Permission]struct{}
}

// NewStaticChecker, verilen rol→izinler eşlemesinden bir StaticChecker kurar.
func NewStaticChecker(roles map[string][]Permission) *StaticChecker {
	m := make(map[string]map[Permission]struct{}, len(roles))
	for role, perms := range roles {
		set := make(map[Permission]struct{}, len(perms))
		for _, p := range perms {
			set[p] = struct{}{}
		}
		m[role] = set
	}
	return &StaticChecker{roles: m}
}

// Allows, rolün belirtilen izne sahip olup olmadığını döner.
func (s *StaticChecker) Allows(_ context.Context, role string, perm Permission) (bool, error) {
	perms, ok := s.roles[role]
	if !ok {
		return false, nil
	}
	_, ok = perms[perm]
	return ok, nil
}

// PermissionsFor, rolün sahip olduğu izinleri sıralı string dilimi olarak döner.
func (s *StaticChecker) PermissionsFor(_ context.Context, role string) ([]string, error) {
	perms, ok := s.roles[role]
	if !ok {
		return []string{}, nil
	}
	out := make([]string, 0, len(perms))
	for p := range perms {
		out = append(out, string(p))
	}
	sort.Strings(out)
	return out, nil
}
