package goui

import (
	"context"

	"github.com/zatrano/gocore/pkg/rbac"
)

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

type dashboardController struct{}

func (c *dashboardController) Mount(context.Context, *Page) error { return nil }

func (c *dashboardController) Render(p *Page) (string, error) {
	ctaHref, ctaLabel := "/dashboard/account", "Hesabım"
	if p.Allowed(context.Background(), rbac.PermUsersList) {
		ctaHref, ctaLabel = "/dashboard/users", "Kullanıcılar"
	} else if p.Allowed(context.Background(), rbac.PermNotificationsSend) {
		ctaHref, ctaLabel = "/dashboard/notifications/send", "Bildirim Gönder"
	}
	return p.RenderView("pages.dashboard", map[string]any{
		"Role":     p.Actor.Role,
		"Email":    p.Actor.Email,
		"CTAHref":  ctaHref,
		"CTALabel": ctaLabel,
	})
}

func (c *dashboardController) HandleEvent(context.Context, *Page, string, map[string]any) error {
	return nil
}
