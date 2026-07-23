package auth

import "context"

// MFARepository, TOTP tekrar kullanımını ve tek kullanımlık kurtarma
// kodlarını atomik olarak yöneten persistence portudur.
type MFARepository interface {
	ConsumeTOTPStep(ctx context.Context, userID string, step int64) (bool, error)
	ResetTOTPStep(ctx context.Context, userID string) error
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	ConsumeRecoveryCode(ctx context.Context, userID, hash string) (bool, error)
	DeleteRecoveryCodes(ctx context.Context, userID string) error
}
