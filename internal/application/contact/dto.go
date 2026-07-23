package contact

import (
	"github.com/zatrano/gocore/internal/domain/contact"
	"github.com/zatrano/gocore/pkg/datetime"
)

// View, iletişim mesajının okuma-tarafı DTO'sudur.
type View struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Email     string             `json:"email"`
	Message   string             `json:"message"`
	Locale    string             `json:"locale"`
	Status    string             `json:"status"`
	Read      bool               `json:"read"`
	CreatedAt datetime.JSONTime  `json:"created_at"`
	ReadAt    *datetime.JSONTime `json:"read_at,omitempty"`
}

func toView(m *contact.Message) View {
	return View{
		ID:        m.ID().String(),
		Name:      m.Name(),
		Email:     m.Email().String(),
		Message:   m.Body(),
		Locale:    m.Locale(),
		Status:    string(m.Status()),
		Read:      m.IsRead(),
		CreatedAt: datetime.FromTime(m.CreatedAt()),
		ReadAt:    datetime.PtrFromTime(m.ReadAt()),
	}
}

func toViews(items []*contact.Message) []View {
	views := make([]View, 0, len(items))
	for _, m := range items {
		views = append(views, toView(m))
	}
	return views
}
