package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/zatrano/gocore/internal/application/authz"
	appuser "github.com/zatrano/gocore/internal/application/user"
)

func (g *graph) wireAuthz(ctx context.Context) error {
	g.authzResolver = authz.NewResolver(g.roleRepo, time.Minute)
	if err := authz.NewSyncer(g.roleRepo, g.txManager).Sync(ctx); err != nil {
		return fmt.Errorf("bootstrap: rbac senkronu: %w", err)
	}
	seeded, err := g.roleRepo.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap: izin listesi: %w", err)
	}
	if len(seeded) == 0 {
		return fmt.Errorf("bootstrap: permissions tablosu boş — önce `go run ./cmd/migrate up` çalıştırın")
	}
	g.authzResolver.Invalidate()
	g.authzService = authz.NewService(g.roleRepo, g.authzResolver, g.txManager, g.publisher)
	g.roleChecker = authz.NewRoleExistsChecker(g.roleRepo)
	g.userAccess = appuser.NewAccess(g.authzResolver)
	return nil
}
