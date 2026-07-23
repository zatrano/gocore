package auth

import "context"

// TokenRepository, OneTimeToken aggregate'i için persistence portudur.
type TokenRepository interface {
	Save(ctx context.Context, token *OneTimeToken) error
	FindByHash(ctx context.Context, tokenHash string) (*OneTimeToken, error)
	MarkUsed(ctx context.Context, id TokenID) (bool, error)
	DeleteForUser(ctx context.Context, userID string, purpose TokenPurpose) error
	DeleteExpired(ctx context.Context) error
}
