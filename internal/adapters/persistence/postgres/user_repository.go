// Package postgres, domain repository portlarının PostgreSQL (pgx + sqlc)
// implementasyonlarını içerir.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/fieldenc"
	"github.com/zatrano/gocore/pkg/pagination"
)

// UserRepository, user.Repository portunun PostgreSQL implementasyonudur.
// Tüm sorgular sqlc ile üretilen type-safe koddan geçer; TxManager sayesinde
// aynı repo hem havuzla hem de aktif transaction içinde çalışır.
type UserRepository struct {
	tx     *database.TxManager
	fields *fieldenc.Cipher
}

// NewUserRepository, repository'yi TxManager ile kurar.
func NewUserRepository(tx *database.TxManager, fields ...*fieldenc.Cipher) *UserRepository {
	var cipher *fieldenc.Cipher
	if len(fields) > 0 {
		cipher = fields[0]
	}
	return &UserRepository{tx: tx, fields: cipher}
}

// queries, context'teki aktif transaction'a (yoksa havuza) bağlı Queries döner.
func (r *UserRepository) queries(ctx context.Context) *db.Queries {
	return db.New(r.tx.DB(ctx))
}

// EncryptLegacyMFASecrets, önceki sürümlerde düz metin saklanan TOTP
// secret'larını idempotent olarak şifreler.
func (r *UserRepository) EncryptLegacyMFASecrets(ctx context.Context) error {
	if r.fields == nil || !r.fields.Enabled() {
		return nil
	}
	rows, err := r.tx.DB(ctx).Query(ctx, `
		SELECT id, mfa_secret
		FROM users
		WHERE mfa_secret <> '' AND mfa_secret NOT LIKE 'enc:v1:%'
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type legacySecret struct {
		id     uuid.UUID
		secret string
	}
	var legacy []legacySecret
	for rows.Next() {
		var item legacySecret
		if err := rows.Scan(&item.id, &item.secret); err != nil {
			return err
		}
		legacy = append(legacy, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range legacy {
		encrypted, err := r.fields.Encrypt(item.secret)
		if err != nil {
			return err
		}
		if _, err := r.tx.DB(ctx).Exec(ctx,
			`UPDATE users SET mfa_secret = $2 WHERE id = $1 AND mfa_secret = $3`,
			item.id, encrypted, item.secret); err != nil {
			return err
		}
	}
	return nil
}

// pgUniqueViolation, PostgreSQL unique constraint ihlali kodu.
const pgUniqueViolation = "23505"

// Save, yeni kullanıcıyı kalıcılaştırır.
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	mfaSecret, err := r.fields.Encrypt(u.MFASecret())
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).CreateUser(ctx, db.CreateUserParams{
		ID:              u.ID().UUID(),
		Email:           u.Email().String(),
		Phone:           u.Phone().String(),
		Name:            u.Name(),
		PasswordHash:    u.Password().Encoded(),
		Role:            u.Role().String(),
		Active:          u.IsActive(),
		EmailVerified:   u.IsEmailVerified(),
		MfaEnabled:      u.MFAEnabled(),
		MfaSecret:       mfaSecret,
		PreferredLocale: u.PreferredLocale().String(),
		CreatedAt:       ts(u.CreatedAt()),
		UpdatedAt:       ts(u.UpdatedAt()),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return user.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// Update, kullanıcıyı optimistic locking ile günceller.
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	mfaSecret, err := r.fields.Encrypt(u.MFASecret())
	if err != nil {
		return err
	}
	_, err = r.queries(ctx).UpdateUser(ctx, db.UpdateUserParams{
		ID:              u.ID().UUID(),
		Email:           u.Email().String(),
		Phone:           u.Phone().String(),
		Name:            u.Name(),
		PasswordHash:    u.Password().Encoded(),
		Role:            u.Role().String(),
		Active:          u.IsActive(),
		EmailVerified:   u.IsEmailVerified(),
		MfaEnabled:      u.MFAEnabled(),
		MfaSecret:       mfaSecret,
		PreferredLocale: u.PreferredLocale().String(),
		UpdatedAt:       ts(u.UpdatedAt()),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return user.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

// FindByID, canlı kullanıcıyı kimliğine göre getirir.
func (r *UserRepository) FindByID(ctx context.Context, id user.ID) (*user.User, error) {
	row, err := r.queries(ctx).GetUserByID(ctx, id.UUID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return toDomain(row, r.fields)
}

// FindByIDIncludeDeleted, silinmiş dahil kullanıcıyı getirir.
func (r *UserRepository) FindByIDIncludeDeleted(ctx context.Context, id user.ID) (*user.User, error) {
	row, err := r.queries(ctx).GetUserByIDAny(ctx, id.UUID())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return toDomain(row, r.fields)
}

// FindByEmail, kullanıcıyı e-postaya göre getirir.
func (r *UserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	row, err := r.queries(ctx).GetUserByEmail(ctx, email.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return toDomain(row, r.fields)
}

// FindByIDs, birden fazla kullanıcıyı TEK sorguda getirir (batch → N+1 önleme).
func (r *UserRepository) FindByIDs(ctx context.Context, ids []user.ID) ([]*user.User, error) {
	if len(ids) == 0 {
		return []*user.User{}, nil
	}
	uuids := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuids[i] = id.UUID()
	}
	rows, err := r.queries(ctx).GetUsersByIDs(ctx, uuids)
	if err != nil {
		return nil, err
	}
	return toDomainSlice(rows, r.fields)
}

// ExistsByEmail, e-posta kayıtlı mı diye kontrol eder.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email user.Email) (bool, error) {
	return r.queries(ctx).ExistsUserByEmail(ctx, email.String())
}

// Delete, kullanıcıyı yazılımsal olarak siler (soft delete): deleted_at damgalar.
func (r *UserRepository) Delete(ctx context.Context, id user.ID) error {
	n, err := r.queries(ctx).SoftDeleteUser(ctx, id.UUID())
	if err != nil {
		return err
	}
	if n == 0 {
		// Kayıt yok ya da zaten silinmiş.
		return user.ErrNotFound
	}
	return nil
}

// Restore, yazılımsal silmeyi geri alır.
func (r *UserRepository) Restore(ctx context.Context, id user.ID) error {
	n, err := r.queries(ctx).RestoreUser(ctx, id.UUID())
	if err != nil {
		return err
	}
	if n == 0 {
		// Kayıt yok ya da silinmiş değil.
		return user.ErrNotFound
	}
	return nil
}

// HardDelete, soft-delete edilmiş bir kaydı kalıcı olarak siler.
func (r *UserRepository) HardDelete(ctx context.Context, id user.ID) error {
	n, err := r.queries(ctx).HardDeleteUser(ctx, id.UUID())
	if err != nil {
		return err
	}
	if n == 0 {
		return user.ErrNotFound
	}
	return nil
}

// List, offset veya keyset (Cursor) sayfalama ile kullanıcıları listeler.
func (r *UserRepository) List(
	ctx context.Context, filter user.ListFilter, page pagination.Request,
) (pagination.Page[*user.User], error) {
	limit := pagination.NormalizeLimit(page.Limit)
	pageNum := page.Page
	if pageNum < 1 {
		pageNum = 1
	}
	if page.Cursor != "" {
		pageNum = 1
	}

	role := optText(filter.Role)
	active := optBool(filter.Active)
	search := optSearch(filter.Search)
	deleted := optDeletedFilter(filter.Deleted)

	countParams := db.CountUsersParams{
		Role: role, Active: active, Search: search, Deleted: deleted,
	}
	total, err := r.queries(ctx).CountUsers(ctx, countParams)
	if err != nil {
		return pagination.Page[*user.User]{}, err
	}

	limit32 := pagination.LimitInt32(limit)
	var rows []db.User

	if page.Cursor != "" {
		cursorAt, cursorID, decErr := pagination.DecodeCursor(page.Cursor)
		if decErr != nil {
			return pagination.Page[*user.User]{}, decErr
		}
		cursorUUID, parseErr := uuid.Parse(cursorID)
		if parseErr != nil {
			return pagination.Page[*user.User]{}, pagination.ErrInvalidCursor
		}
		cursorTS := ts(cursorAt)
		keysetBase := db.ListUsersDescKeysetParams{
			Deleted: deleted, Role: role, Active: active, Search: search,
			CursorCreatedAt: cursorTS, CursorID: cursorUUID, Lmt: limit32,
		}
		if page.Ascending {
			rows, err = r.queries(ctx).ListUsersAscKeyset(ctx, db.ListUsersAscKeysetParams(keysetBase))
		} else {
			rows, err = r.queries(ctx).ListUsersDescKeyset(ctx, keysetBase)
		}
	} else {
		off := pagination.OffsetInt32(pageNum, limit)
		if page.Ascending {
			rows, err = r.queries(ctx).ListUsersAscOffset(ctx, db.ListUsersAscOffsetParams{
				Role: role, Active: active, Search: search, Deleted: deleted,
				Lmt: limit32, Off: off,
			})
		} else {
			rows, err = r.queries(ctx).ListUsersDescOffset(ctx, db.ListUsersDescOffsetParams{
				Role: role, Active: active, Search: search, Deleted: deleted,
				Lmt: limit32, Off: off,
			})
		}
	}
	if err != nil {
		return pagination.Page[*user.User]{}, err
	}

	users, err := toDomainSlice(rows, r.fields)
	if err != nil {
		return pagination.Page[*user.User]{}, err
	}
	out := pagination.NewPage(users, pageNum, limit, total)
	out.NextCursor = pagination.EncodeNextCursor(users, limit, func(u *user.User) (time.Time, string) {
		return u.CreatedAt(), u.ID().String()
	})
	return out, nil
}

// CountActiveByRole, canlı kullanıcıları role göre sayar.
func (r *UserRepository) CountActiveByRole(ctx context.Context, role user.Role) (int64, error) {
	return r.queries(ctx).CountActiveUsersByRole(ctx, role.String())
}

var _ user.Repository = (*UserRepository)(nil)
