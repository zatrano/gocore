package notification

import (
	"context"
	"errors"
	"strings"

	duser "github.com/zatrano/gocore/internal/domain/user"
)

// RecipientResolver, kanal türüne göre alıcıyı gönderim için hazırlar.
type RecipientResolver interface {
	Resolve(ctx context.Context, channel Channel, r Recipient) (Recipient, error)
}

type noopResolver struct{}

func (noopResolver) Resolve(_ context.Context, _ Channel, r Recipient) (Recipient, error) {
	return r, nil
}

// UserRepoResolver, in-app alıcıları kullanıcı kimliği veya e-posta ile çözer.
type UserRepoResolver struct {
	Users duser.Repository
}

// Resolve, in-app için aktif kullanıcı kimliğini garanti eder; e-posta kanalında
// bilinen kullanıcıların tercih dilini doldurur (toplu dil filtresi için).
func (r UserRepoResolver) Resolve(ctx context.Context, channel Channel, recip Recipient) (Recipient, error) {
	switch channel {
	case ChannelInApp:
		return r.resolveInApp(ctx, recip)
	case ChannelEmail:
		return r.resolveEmailLocale(ctx, recip)
	default:
		return recip, nil
	}
}

func (r UserRepoResolver) resolveInApp(ctx context.Context, recip Recipient) (Recipient, error) {
	if id := strings.TrimSpace(recip.UserID); id != "" {
		parsed, err := duser.ParseID(id)
		if err != nil {
			return Recipient{}, ErrRecipientRequired
		}
		u, err := r.Users.FindByID(ctx, parsed)
		if err != nil {
			if errors.Is(err, duser.ErrNotFound) {
				return Recipient{}, ErrRecipientNotFound
			}
			return Recipient{}, err
		}
		if !u.IsActive() {
			return Recipient{}, ErrRecipientNotFound
		}
		recip.UserID = u.ID().String()
		if recip.Email == "" {
			recip.Email = u.Email().String()
		}
		recip.Locale = u.PreferredLocale().String()
		return recip, nil
	}
	if strings.TrimSpace(recip.Email) == "" {
		return Recipient{}, ErrRecipientRequired
	}
	u, err := r.findActiveByEmail(ctx, recip.Email)
	if err != nil {
		return Recipient{}, err
	}
	recip.UserID = u.ID().String()
	recip.Locale = u.PreferredLocale().String()
	return recip, nil
}

// resolveEmailLocale, kayıtlı kullanıcıysa tercih dilini yazar; değilse alıcıyı olduğu gibi bırakır.
func (r UserRepoResolver) resolveEmailLocale(ctx context.Context, recip Recipient) (Recipient, error) {
	if strings.TrimSpace(recip.Email) == "" {
		return recip, nil
	}
	u, err := r.findActiveByEmail(ctx, recip.Email)
	if err != nil {
		if errors.Is(err, ErrRecipientNotFound) {
			return recip, nil
		}
		return Recipient{}, err
	}
	recip.Locale = u.PreferredLocale().String()
	return recip, nil
}

func (r UserRepoResolver) findActiveByEmail(ctx context.Context, raw string) (*duser.User, error) {
	email, err := duser.NewEmail(raw)
	if err != nil {
		return nil, ErrRecipientNotFound
	}
	u, err := r.Users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, duser.ErrNotFound) {
			return nil, ErrRecipientNotFound
		}
		return nil, err
	}
	if !u.IsActive() {
		return nil, ErrRecipientNotFound
	}
	return u, nil
}
