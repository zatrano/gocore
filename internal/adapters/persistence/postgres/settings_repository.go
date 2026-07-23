package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

// SettingsRepository, domain/settings.Repository portunun PostgreSQL implementasyonudur.
type SettingsRepository struct {
	tx *database.TxManager
}

// NewSettingsRepository, repository'yi kurar.
func NewSettingsRepository(tx *database.TxManager) *SettingsRepository {
	return &SettingsRepository{tx: tx}
}

func (r *SettingsRepository) queries(ctx context.Context) *db.Queries {
	return db.New(r.tx.DB(ctx))
}

// Get, anahtara göre ayar değerini döner; yoksa boş string.
func (r *SettingsRepository) Get(ctx context.Context, key domainsettings.SettingKey) (string, error) {
	v, err := r.queries(ctx).GetSetting(ctx, key.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

// Set, ayar değerini upsert eder.
func (r *SettingsRepository) Set(ctx context.Context, key domainsettings.SettingKey, value string) error {
	return r.queries(ctx).UpsertSetting(ctx, db.UpsertSettingParams{
		Key: key.String(), Value: value,
	})
}

var _ domainsettings.Repository = (*SettingsRepository)(nil)
