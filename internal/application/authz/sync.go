package authz

import (
	"context"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
	"github.com/zatrano/gocore/pkg/rbac"
)

// Syncer, sistem rollerini ve admin izin atamasını veritabanıyla uyumlar.
type Syncer struct {
	repo domainrbac.Repository
	tx   appshared.TxManager
}

// NewSyncer, Syncer'ı kurar.
func NewSyncer(repo domainrbac.Repository, tx appshared.TxManager) *Syncer {
	return &Syncer{repo: repo, tx: tx}
}

// Sync, uygulama açılışında çağrılır. Tek transaction içinde:
//   - sistem rollerinin (admin/user) varlığını garanti eder,
//   - admin rolüne DB'deki tüm izinleri atar (yeni migration izinleri dahil).
func (s *Syncer) Sync(ctx context.Context) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.ensurePermissionCatalog(ctx); err != nil {
			return err
		}
		if err := s.ensureSystemRole(ctx, domainrbac.RoleAdmin, "Tüm yetkilere sahip sistem yöneticisi"); err != nil {
			return err
		}
		if err := s.ensureSystemRole(ctx, domainrbac.RoleUser, "Standart kullanıcı"); err != nil {
			return err
		}
		return s.repo.GrantAllPermissions(ctx, domainrbac.RoleAdmin)
	})
}

func (s *Syncer) ensurePermissionCatalog(ctx context.Context) error {
	for _, def := range rbac.Catalog() {
		pn, err := domainrbac.ParsePermissionName(string(def.Permission))
		if err != nil {
			return err
		}
		exists, err := s.repo.PermissionExists(ctx, pn)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		perm, err := domainrbac.NewPermission(string(def.Permission), def.Description)
		if err != nil {
			return err
		}
		if err := s.repo.CreatePermission(ctx, perm); err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) ensureSystemRole(ctx context.Context, name domainrbac.RoleName, description string) error {
	exists, err := s.repo.Exists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	role, err := domainrbac.CreateSystemRole(name, description)
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, role)
}
