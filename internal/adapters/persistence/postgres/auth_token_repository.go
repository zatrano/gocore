package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

// AuthTokenRepository, domain/auth.TokenRepository portunun PostgreSQL implementasyonudur.
type AuthTokenRepository struct {
	tx *database.TxManager
}

// NewAuthTokenRepository, repository'yi TxManager ile kurar.
func NewAuthTokenRepository(tx *database.TxManager) *AuthTokenRepository {
	return &AuthTokenRepository{tx: tx}
}

func (r *AuthTokenRepository) queries(ctx context.Context) *db.Queries {
	return db.New(r.tx.DB(ctx))
}

// Save, yeni bir token kaydı ekler.
func (r *AuthTokenRepository) Save(ctx context.Context, token *domainauth.OneTimeToken) error {
	uid, err := uuid.Parse(token.UserID())
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).CreateAuthToken(ctx, db.CreateAuthTokenParams{
		ID:        token.ID().UUID(),
		UserID:    uid,
		Purpose:   token.Purpose().String(),
		TokenHash: token.TokenHash(),
		ExpiresAt: ts(token.ExpiresAt()),
	})
	return err
}

// FindByHash, token özetine göre kaydı getirir.
func (r *AuthTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domainauth.OneTimeToken, error) {
	row, err := r.queries(ctx).GetAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainauth.ErrInvalidToken
		}
		return nil, err
	}
	return domainauth.Rehydrate(
		domainauth.TokenIDFromUUID(row.ID),
		row.UserID.String(),
		domainauth.TokenPurpose(row.Purpose),
		row.TokenHash,
		row.ExpiresAt.Time,
		row.UsedAt.Valid,
	), nil
}

// MarkUsed, token'ı tek kullanımlık olarak işaretler.
func (r *AuthTokenRepository) MarkUsed(ctx context.Context, id domainauth.TokenID) (bool, error) {
	rows, err := r.queries(ctx).MarkAuthTokenUsed(ctx, id.UUID())
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

// DeleteForUser, aynı amaçlı önceki token'ları siler.
func (r *AuthTokenRepository) DeleteForUser(ctx context.Context, userID string, purpose domainauth.TokenPurpose) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	return r.queries(ctx).DeleteAuthTokensForUser(ctx, db.DeleteAuthTokensForUserParams{
		UserID:  uid,
		Purpose: purpose.String(),
	})
}

// DeleteExpired, süresi dolmuş/kullanılmış token'ları temizler.
func (r *AuthTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.queries(ctx).DeleteExpiredAuthTokens(ctx)
}

var _ domainauth.TokenRepository = (*AuthTokenRepository)(nil)
