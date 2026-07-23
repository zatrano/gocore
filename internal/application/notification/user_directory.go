package notification

import (
	"context"

	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/pagination"
)

// UserRepoDirectory, user.Repository üzerinden aktif kullanıcıları listeler.
type UserRepoDirectory struct {
	Repo user.Repository
}

// ListActiveContacts, soft-delete edilmemiş aktif kullanıcıları sayfalar.
func (d UserRepoDirectory) ListActiveContacts(ctx context.Context, pageNum int, limit int) ([]UserContact, bool, error) {
	active := true
	page, err := d.Repo.List(ctx, user.ListFilter{Active: &active}, pagination.Request{
		Page: pageNum, Limit: limit,
	})
	if err != nil {
		return nil, false, err
	}
	out := make([]UserContact, 0, len(page.Items))
	for _, u := range page.Items {
		out = append(out, UserContact{
			ID:     u.ID().String(),
			Email:  u.Email().String(),
			Phone:  u.Phone().String(),
			Locale: u.PreferredLocale().String(),
		})
	}
	return out, page.HasNext(), nil
}
