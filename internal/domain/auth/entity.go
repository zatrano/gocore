package auth

import (
	"time"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// OneTimeToken, e-posta doğrulama ve şifre sıfırlama gibi akışlarda kullanılan
// tek kullanımlık token aggregate'idir. Veritabanında yalnızca hash saklanır.
type OneTimeToken struct {
	shared.EventRecorder

	id        TokenID
	userID    string
	purpose   TokenPurpose
	tokenHash string
	expiresAt time.Time
	used      bool
}

// NewOneTimeToken, yeni bir tek kullanımlık token oluşturur.
func NewOneTimeToken(userID string, purpose TokenPurpose, tokenHash string, expiresAt time.Time) (*OneTimeToken, error) {
	if userID == "" {
		return nil, ErrInvalidToken
	}
	if tokenHash == "" {
		return nil, ErrInvalidToken
	}
	if !expiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidToken
	}
	t := &OneTimeToken{
		id:        NewTokenID(),
		userID:    userID,
		purpose:   purpose,
		tokenHash: tokenHash,
		expiresAt: expiresAt.UTC(),
	}
	t.Record(OneTimeTokenCreatedEvent{
		BaseEvent: shared.NewBaseEvent(EventOneTimeTokenCreated, t.id.String()),
		UserID:    userID,
		Purpose:   purpose.String(),
	})
	return t, nil
}

// Rehydrate, persistence katmanından okunan token'ı yeniden oluşturur.
func Rehydrate(id TokenID, userID string, purpose TokenPurpose, tokenHash string, expiresAt time.Time, used bool) *OneTimeToken {
	return &OneTimeToken{
		id:        id,
		userID:    userID,
		purpose:   purpose,
		tokenHash: tokenHash,
		expiresAt: expiresAt.UTC(),
		used:      used,
	}
}

// Consume, token'ın belirtilen amaç için geçerli olduğunu doğrular ve tüketir.
func (t *OneTimeToken) Consume(expectedPurpose TokenPurpose, now time.Time) error {
	if err := t.validate(expectedPurpose, now); err != nil {
		return err
	}
	t.used = true
	t.Record(OneTimeTokenUsedEvent{
		BaseEvent: shared.NewBaseEvent(EventOneTimeTokenUsed, t.id.String()),
		UserID:    t.userID,
		Purpose:   t.purpose.String(),
	})
	return nil
}

func (t *OneTimeToken) validate(expectedPurpose TokenPurpose, now time.Time) error {
	if t.purpose != expectedPurpose {
		return ErrInvalidToken
	}
	if t.used {
		return ErrInvalidToken
	}
	if !now.Before(t.expiresAt) {
		return ErrInvalidToken
	}
	return nil
}

func (t *OneTimeToken) ID() TokenID           { return t.id }
func (t *OneTimeToken) UserID() string        { return t.userID }
func (t *OneTimeToken) Purpose() TokenPurpose { return t.purpose }
func (t *OneTimeToken) TokenHash() string     { return t.tokenHash }
func (t *OneTimeToken) ExpiresAt() time.Time  { return t.expiresAt }
func (t *OneTimeToken) Used() bool            { return t.used }
