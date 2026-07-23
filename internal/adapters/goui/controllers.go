package goui

import (
	"context"
	"fmt"
)

func controllerFor(screen string) Controller {
	for _, factory := range []func(string) Controller{
		publicAuthController,
		accountUsersController,
		contactsController,
		rbacNotificationsController,
		settingsPaymentsAuditController,
	} {
		if controller := factory(screen); controller != nil {
			return controller
		}
	}
	return &missingController{screen: screen}
}

type missingController struct {
	screen string
}

func (*missingController) Mount(context.Context, *Page) error { return nil }

func (c *missingController) Render(p *Page) (string, error) {
	return p.RenderView("pages.missing", map[string]any{
		"Message": fmt.Sprintf("%q bileşeni kayıtlı değil", c.screen),
	})
}

func (*missingController) HandleEvent(context.Context, *Page, string, map[string]any) error {
	return nil
}
