// Package user (application), User bounded context için CQRS use-case'lerini
// içerir: command'ler (yazma) ve query'ler (okuma) ayrıştırılmıştır.
package user

import (
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/datetime"
)

// View, bir kullanıcının okuma-tarafı (read model) temsilidir. Immutable bir
// DTO'dur: yalnızca kurucusuyla üretilir, alanları dışarıdan değiştirilmez ve
// asla hassas veri (şifre hash'i) içermez.
type View struct {
	ID              string             `json:"id"`
	Email           string             `json:"email"`
	Phone           string             `json:"phone"`
	Name            string             `json:"name"`
	Role            string             `json:"role"`
	Active          bool               `json:"active"`
	EmailVerified   bool               `json:"email_verified"`
	MFAEnabled      bool               `json:"mfa_enabled"`
	PreferredLocale string             `json:"preferred_locale"`
	CreatedAt       datetime.JSONTime  `json:"created_at"`
	UpdatedAt       datetime.JSONTime  `json:"updated_at"`
	Deleted         bool               `json:"deleted"`
	DeletedAt       *datetime.JSONTime `json:"deleted_at,omitempty"`
}

// newView, domain aggregate'inden güvenli bir View üretir (şifre hariç).
func newView(u *user.User) View {
	return View{
		ID:              u.ID().String(),
		Email:           u.Email().String(),
		Phone:           u.Phone().String(),
		Name:            u.Name(),
		Role:            u.Role().String(),
		Active:          u.IsActive(),
		EmailVerified:   u.IsEmailVerified(),
		MFAEnabled:      u.MFAEnabled(),
		PreferredLocale: u.PreferredLocale().String(),
		CreatedAt:       datetime.FromTime(u.CreatedAt()),
		UpdatedAt:       datetime.FromTime(u.UpdatedAt()),
		Deleted:         u.IsDeleted(),
		DeletedAt:       datetime.PtrFromTime(u.DeletedAt()),
	}
}

// newViews, birden fazla aggregate'i View listesine çevirir.
func newViews(users []*user.User) []View {
	views := make([]View, 0, len(users))
	for _, u := range users {
		views = append(views, newView(u))
	}
	return views
}

// MeView, oturum açmış kullanıcının profili + sahip olduğu izinler.
type MeView struct {
	View
	Permissions []string `json:"permissions"`
}
