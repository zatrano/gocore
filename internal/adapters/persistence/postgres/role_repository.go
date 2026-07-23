package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

// RoleRepository, domain/rbac.Repository portunun PostgreSQL implementasyonudur.
type RoleRepository struct {
	tx *database.TxManager
}

// NewRoleRepository, repository'yi TxManager ile kurar.
func NewRoleRepository(tx *database.TxManager) *RoleRepository {
	return &RoleRepository{tx: tx}
}

func (r *RoleRepository) queries(ctx context.Context) *db.Queries {
	return db.New(r.tx.DB(ctx))
}

func (r *RoleRepository) loadPermissions(ctx context.Context, name domainrbac.RoleName) ([]domainrbac.PermissionName, error) {
	raw, err := r.queries(ctx).ListPermissionsForRole(ctx, name.String())
	if err != nil {
		return nil, err
	}
	out := make([]domainrbac.PermissionName, len(raw))
	for i, p := range raw {
		out[i] = domainrbac.PermissionName(p)
	}
	return out, nil
}

func (r *RoleRepository) Save(ctx context.Context, role *domainrbac.Role) error {
	return r.queries(ctx).CreateRole(ctx, db.CreateRoleParams{
		ID:          uuid.New(),
		Name:        role.Name().String(),
		Description: role.Description(),
		IsSystem:    role.IsSystem(),
	})
}

func (r *RoleRepository) UpdateDescription(ctx context.Context, name domainrbac.RoleName, description string) error {
	n, err := r.queries(ctx).UpdateRoleDescription(ctx, db.UpdateRoleDescriptionParams{
		Name:        name.String(),
		Description: description,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return domainrbac.ErrRoleNotFound
	}
	return nil
}

func (r *RoleRepository) ReplacePermissions(ctx context.Context, name domainrbac.RoleName, perms []domainrbac.PermissionName) error {
	q := r.queries(ctx)
	if err := q.ClearRolePermissions(ctx, name.String()); err != nil {
		return err
	}
	for _, p := range perms {
		if err := q.AddRolePermission(ctx, db.AddRolePermissionParams{
			Name: name.String(), Name_2: p.String(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *RoleRepository) FindByName(ctx context.Context, name domainrbac.RoleName) (*domainrbac.Role, error) {
	row, err := r.queries(ctx).GetRoleByName(ctx, name.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainrbac.ErrRoleNotFound
		}
		return nil, err
	}
	perms, err := r.loadPermissions(ctx, name)
	if err != nil {
		return nil, err
	}
	return domainrbac.Rehydrate(
		domainrbac.RoleName(row.Name),
		row.Description,
		row.IsSystem,
		perms,
	), nil
}

func (r *RoleRepository) List(ctx context.Context) ([]*domainrbac.Role, error) {
	rows, err := r.queries(ctx).ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*domainrbac.Role, 0, len(rows))
	for _, row := range rows {
		name := domainrbac.RoleName(row.Name)
		perms, err := r.loadPermissions(ctx, name)
		if err != nil {
			return nil, err
		}
		out = append(out, domainrbac.Rehydrate(name, row.Description, row.IsSystem, perms))
	}
	return out, nil
}

func (r *RoleRepository) Delete(ctx context.Context, name domainrbac.RoleName) error {
	n, err := r.queries(ctx).DeleteRole(ctx, name.String())
	if err != nil {
		return err
	}
	if n == 0 {
		return domainrbac.ErrRoleNotFound
	}
	return nil
}

func (r *RoleRepository) Exists(ctx context.Context, name domainrbac.RoleName) (bool, error) {
	return r.queries(ctx).RoleExists(ctx, name.String())
}

func (r *RoleRepository) CountUsersWithRole(ctx context.Context, name domainrbac.RoleName) (int64, error) {
	return r.queries(ctx).CountUsersWithRole(ctx, name.String())
}

func (r *RoleRepository) GrantAllPermissions(ctx context.Context, name domainrbac.RoleName) error {
	return r.queries(ctx).GrantAllPermissionsToRole(ctx, name.String())
}

func (r *RoleRepository) CreatePermission(ctx context.Context, perm domainrbac.Permission) error {
	return r.queries(ctx).CreatePermission(ctx, db.CreatePermissionParams{
		ID:          uuid.New(),
		Name:        perm.Name().String(),
		Description: perm.Description(),
	})
}

func (r *RoleRepository) PermissionExists(ctx context.Context, name domainrbac.PermissionName) (bool, error) {
	return r.queries(ctx).PermissionExists(ctx, name.String())
}

func (r *RoleRepository) UpdatePermissionDescription(ctx context.Context, name domainrbac.PermissionName, description string) error {
	_, err := r.queries(ctx).UpdatePermissionDescription(ctx, db.UpdatePermissionDescriptionParams{
		Name: name.String(), Description: description,
	})
	return err
}

func (r *RoleRepository) ListPermissions(ctx context.Context) ([]domainrbac.Permission, error) {
	rows, err := r.queries(ctx).ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domainrbac.Permission, 0, len(rows))
	for _, row := range rows {
		out = append(out, domainrbac.RehydratePermission(
			domainrbac.PermissionName(row.Name),
			row.Description,
		))
	}
	return out, nil
}

var _ domainrbac.Repository = (*RoleRepository)(nil)
