package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/database"
)

// MFARepository, MFA replay ve recovery-code işlemlerini atomik SQL ile yürütür.
type MFARepository struct {
	tx *database.TxManager
}

func NewMFARepository(tx *database.TxManager) *MFARepository {
	return &MFARepository{tx: tx}
}

func (r *MFARepository) ConsumeTOTPStep(ctx context.Context, userID string, step int64) (bool, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	tag, err := r.tx.DB(ctx).Exec(ctx, `
		UPDATE users
		SET mfa_last_used_step = $2, updated_at = now()
		WHERE id = $1
		  AND mfa_enabled = TRUE
		  AND (mfa_last_used_step IS NULL OR mfa_last_used_step < $2)
	`, id, step)
	return tag.RowsAffected() == 1, err
}

func (r *MFARepository) ResetTOTPStep(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx,
		`UPDATE users SET mfa_last_used_step = NULL WHERE id = $1`, id)
	return err
}

func (r *MFARepository) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	db := r.tx.DB(ctx)
	if _, err := db.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, id); err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err := db.Exec(ctx, `
			INSERT INTO mfa_recovery_codes (id, user_id, code_hash)
			VALUES ($1, $2, $3)
		`, uuid.New(), id, hash); err != nil {
			return err
		}
	}
	return nil
}

func (r *MFARepository) ConsumeRecoveryCode(ctx context.Context, userID, hash string) (bool, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return false, err
	}
	var codeID uuid.UUID
	err = r.tx.DB(ctx).QueryRow(ctx, `
		UPDATE mfa_recovery_codes
		SET used_at = now()
		WHERE id = (
			SELECT id FROM mfa_recovery_codes
			WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id
	`, id, hash).Scan(&codeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *MFARepository) DeleteRecoveryCodes(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, id)
	return err
}

var _ appauth.MFARepository = (*MFARepository)(nil)
