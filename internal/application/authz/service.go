package authz

import (
	"context"
	"strings"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
	dshared "github.com/zatrano/gocore/internal/domain/shared"
)

// Service, rol ve izinlerin çalışma zamanı yönetimini sağlar (admin API'si).
// Domain aggregate'leri üzerinden iş kurallarını uygular; kalıcılık
// domain/rbac.Repository portu ile yapılır.
type Service struct {
	repo      domainrbac.Repository
	resolver  *Resolver
	tx        appshared.TxManager
	publisher appshared.EventPublisher
}

// NewService, Service'i kurar.
func NewService(repo domainrbac.Repository, resolver *Resolver, tx appshared.TxManager, publisher appshared.EventPublisher) *Service {
	return &Service{repo: repo, resolver: resolver, tx: tx, publisher: publisher}
}

// ListPermissions, veritabanındaki izinleri döner.
func (s *Service) ListPermissions(ctx context.Context) ([]PermissionInfo, error) {
	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PermissionInfo, len(perms))
	for i, p := range perms {
		out[i] = toPermissionInfo(p)
	}
	return out, nil
}

// CreatePermission, yeni izin kaydı oluşturur.
func (s *Service) CreatePermission(ctx context.Context, name, description string) (PermissionInfo, error) {
	perm, err := domainrbac.NewPermission(name, description)
	if err != nil {
		return PermissionInfo{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		exists, err := s.repo.PermissionExists(ctx, perm.Name())
		if err != nil {
			return err
		}
		if exists {
			return domainrbac.ErrPermissionExists
		}
		return s.repo.CreatePermission(ctx, perm)
	})
	if err != nil {
		return PermissionInfo{}, err
	}
	return toPermissionInfo(perm), nil
}

// UpdatePermission, izin açıklamasını günceller.
func (s *Service) UpdatePermission(ctx context.Context, name, description string) (PermissionInfo, error) {
	pn, err := domainrbac.ParsePermissionName(strings.TrimSpace(name))
	if err != nil {
		return PermissionInfo{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		exists, err := s.repo.PermissionExists(ctx, pn)
		if err != nil {
			return err
		}
		if !exists {
			return domainrbac.ErrPermissionNotFound
		}
		return s.repo.UpdatePermissionDescription(ctx, pn, strings.TrimSpace(description))
	})
	if err != nil {
		return PermissionInfo{}, err
	}
	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return PermissionInfo{}, err
	}
	for _, p := range perms {
		if p.Name() == pn {
			return toPermissionInfo(p), nil
		}
	}
	return PermissionInfo{Name: pn.String(), Description: strings.TrimSpace(description)}, nil
}

// ListRoles, tüm rolleri izinleriyle birlikte döner.
func (s *Service) ListRoles(ctx context.Context) ([]RoleInfo, error) {
	roles, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RoleInfo, len(roles))
	for i, r := range roles {
		out[i] = toRoleInfo(r)
	}
	return out, nil
}

// GetRole, tek bir rolü izinleriyle döner.
func (s *Service) GetRole(ctx context.Context, name string) (RoleInfo, error) {
	rn, err := domainrbac.ParseRoleName(strings.TrimSpace(name))
	if err != nil {
		return RoleInfo{}, err
	}
	role, err := s.repo.FindByName(ctx, rn)
	if err != nil {
		return RoleInfo{}, err
	}
	return toRoleInfo(role), nil
}

// CreateRole, yeni bir rol oluşturur ve (verilmişse) izinlerini atar.
func (s *Service) CreateRole(ctx context.Context, name, description string, perms []string) (RoleInfo, error) {
	role, err := domainrbac.CreateRole(name, description)
	if err != nil {
		return RoleInfo{}, err
	}
	parsed, err := s.parsePermissions(ctx, perms)
	if err != nil {
		return RoleInfo{}, err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		exists, err := s.repo.Exists(ctx, role.Name())
		if err != nil {
			return err
		}
		if exists {
			return domainrbac.ErrRoleExists
		}
		if err := s.repo.Save(ctx, role); err != nil {
			return err
		}
		if len(parsed) > 0 {
			role.ReplacePermissions(parsed)
			if err := s.repo.ReplacePermissions(ctx, role.Name(), parsed); err != nil {
				return err
			}
		}
		return s.publish(ctx, role.PullEvents()...)
	})
	if err != nil {
		return RoleInfo{}, err
	}
	s.resolver.Invalidate()
	return s.GetRole(ctx, role.Name().String())
}

// UpdateRole, rol açıklamasını günceller.
func (s *Service) UpdateRole(ctx context.Context, name, description string) (RoleInfo, error) {
	rn, err := domainrbac.ParseRoleName(strings.TrimSpace(name))
	if err != nil {
		return RoleInfo{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		role, err := s.repo.FindByName(ctx, rn)
		if err != nil {
			return err
		}
		role.UpdateDescription(description)
		if err := s.repo.UpdateDescription(ctx, rn, role.Description()); err != nil {
			return err
		}
		return s.publish(ctx, role.PullEvents()...)
	})
	if err != nil {
		return RoleInfo{}, err
	}
	return s.GetRole(ctx, rn.String())
}

// SetPermissions, rolün izin kümesini tümüyle değiştirir.
func (s *Service) SetPermissions(ctx context.Context, name string, perms []string) (RoleInfo, error) {
	rn, err := domainrbac.ParseRoleName(strings.TrimSpace(name))
	if err != nil {
		return RoleInfo{}, err
	}
	parsed, err := s.parsePermissions(ctx, perms)
	if err != nil {
		return RoleInfo{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		role, err := s.repo.FindByName(ctx, rn)
		if err != nil {
			return err
		}
		role.ReplacePermissions(parsed)
		if err := s.repo.ReplacePermissions(ctx, rn, parsed); err != nil {
			return err
		}
		return s.publish(ctx, role.PullEvents()...)
	})
	if err != nil {
		return RoleInfo{}, err
	}
	s.resolver.Invalidate()
	return s.GetRole(ctx, rn.String())
}

// DeleteRole, sistem-olmayan ve kullanımda olmayan bir rolü siler.
func (s *Service) DeleteRole(ctx context.Context, name string) error {
	rn, err := domainrbac.ParseRoleName(strings.TrimSpace(name))
	if err != nil {
		return err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		role, err := s.repo.FindByName(ctx, rn)
		if err != nil {
			return err
		}
		count, err := s.repo.CountUsersWithRole(ctx, rn)
		if err != nil {
			return err
		}
		if err := role.EnsureDeletable(count); err != nil {
			return err
		}
		role.MarkDeleted()
		if err := s.repo.Delete(ctx, rn); err != nil {
			return err
		}
		return s.publish(ctx, role.PullEvents()...)
	})
	if err != nil {
		return err
	}
	s.resolver.Invalidate()
	return nil
}

func (s *Service) publish(ctx context.Context, events ...dshared.DomainEvent) error {
	if s.publisher == nil || len(events) == 0 {
		return nil
	}
	return s.publisher.Publish(ctx, events...)
}

func (s *Service) parsePermissions(ctx context.Context, perms []string) ([]domainrbac.PermissionName, error) {
	if len(perms) == 0 {
		return nil, nil
	}
	listed, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(listed))
	for _, p := range listed {
		known[p.Name().String()] = struct{}{}
	}
	out := make([]domainrbac.PermissionName, 0, len(perms))
	for _, p := range perms {
		if _, ok := known[p]; !ok {
			return nil, domainrbac.ErrUnknownPermission
		}
		out = append(out, domainrbac.PermissionName(p))
	}
	return out, nil
}
