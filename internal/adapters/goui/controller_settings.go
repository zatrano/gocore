package goui

import (
	"context"
	"errors"

	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

// --- SMS settings ---

type smsSettingsCtrl struct {
	providers []appsettings.SMSProviderRow
	active    string
}

func (c *smsSettingsCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	if p.Deps.Settings == nil {
		return errors.New("settings servisi yapılandırılmamış")
	}
	st := adaptershared.SMSIntegrationStatus(p.Deps.Notify)
	rows, active, err := p.Deps.Settings.ListSMSProviders(ctx, st)
	if err != nil {
		return err
	}
	c.providers, c.active = rows, active
	return nil
}

func (c *smsSettingsCtrl) Render(p *Page) (string, error) {
	type providerRow struct {
		Name         string
		Label        string
		Description  string
		Active       bool
		Configured   bool
		DetailHref   string
		ShowActivate bool
	}
	rows := make([]providerRow, 0, len(c.providers))
	for _, row := range c.providers {
		rows = append(rows, providerRow{
			Name:         row.Name,
			Label:        row.Label,
			Description:  row.Description,
			Active:       row.Active,
			Configured:   row.Configured,
			DetailHref:   "/dashboard/settings/sms/" + row.Name,
			ShowActivate: row.Configured && !row.Active,
		})
	}
	return p.RenderView("pages.sms_settings", map[string]any{
		"Active":    c.active,
		"Providers": rows,
	})
}

func (c *smsSettingsCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	if event != "sms.activate" {
		return nil
	}
	provider := payloadString(payload, "provider")
	if err := validation.Check(p.Deps.Validate, &struct {
		Provider string `validate:"required,oneof=netgsm iletimerkezi"`
	}{Provider: provider}); err != nil {
		return err
	}
	st := adaptershared.SMSIntegrationStatus(p.Deps.Notify)
	if err := adaptershared.ValidateSMSActivation(provider, st); err != nil {
		return err
	}
	if err := p.Deps.Settings.SetSMSActiveProvider(ctx, provider); err != nil {
		return err
	}
	p.Notice = "SMS sağlayıcısı güncellendi"
	p.Redirect = "/dashboard/settings/sms"
	return c.Mount(ctx, p)
}

type smsProviderCtrl struct {
	provider appsettings.SMSProviderRow
}

func (c *smsProviderCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	name := ""
	if p.Params != nil {
		name = p.Params["provider"]
	}
	st := adaptershared.SMSIntegrationStatus(p.Deps.Notify)
	row, err := p.Deps.Settings.GetSMSProvider(ctx, name, st)
	if err != nil {
		return err
	}
	c.provider = row
	return nil
}

func (c *smsProviderCtrl) Render(p *Page) (string, error) {
	return p.RenderView("pages.sms_provider", map[string]any{
		"Name":        c.provider.Name,
		"Label":       c.provider.Label,
		"Description": c.provider.Description,
		"Active":      c.provider.Active,
		"Configured":  c.provider.Configured,
	})
}

func (c *smsProviderCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	if event != "sms.activate" {
		return nil
	}
	provider := payloadString(payload, "provider")
	name := ""
	if p.Params != nil {
		name = p.Params["provider"]
	}
	if err := validation.Check(p.Deps.Validate, &struct {
		Provider string `validate:"required,oneof=netgsm iletimerkezi"`
	}{Provider: provider}); err != nil {
		return err
	}
	if provider != name {
		return domainsettings.ErrInvalidSMSProvider
	}
	st := adaptershared.SMSIntegrationStatus(p.Deps.Notify)
	if err := adaptershared.ValidateSMSActivation(provider, st); err != nil {
		return err
	}
	if err := p.Deps.Settings.SetSMSActiveProvider(ctx, provider); err != nil {
		return err
	}
	p.Notice = "SMS sağlayıcısı güncellendi"
	p.Redirect = "/dashboard/settings/sms"
	return c.Mount(ctx, p)
}

// --- Payment settings ---

type paymentSettingsCtrl struct {
	view appsettings.PaymentView
}

func (c *paymentSettingsCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	st := adaptershared.PaymentIntegrationStatus(p.Deps.Payment)
	view, err := p.Deps.Settings.GetPaymentView(ctx, st)
	if err != nil {
		return err
	}
	c.view = view
	return nil
}

func (c *paymentSettingsCtrl) Render(p *Page) (string, error) {
	type providerRow struct {
		Name         string
		Label        string
		Description  string
		Active       bool
		Configured   bool
		DetailHref   string
		ShowActivate bool
	}
	rows := make([]providerRow, 0, len(c.view.Providers))
	for _, row := range c.view.Providers {
		rows = append(rows, providerRow{
			Name:         row.Name,
			Label:        row.Label,
			Description:  row.Description,
			Active:       row.Active,
			Configured:   row.Configured,
			DetailHref:   "/dashboard/settings/payment/" + row.Name,
			ShowActivate: row.Configured && !row.Active,
		})
	}
	return p.RenderView("pages.payment_settings", map[string]any{
		"Active":    c.view.ActiveProvider,
		"Providers": rows,
	})
}

func (c *paymentSettingsCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	if event != "payment.activate" {
		return nil
	}
	provider := payloadString(payload, "provider")
	if err := validation.Check(p.Deps.Validate, &struct {
		Provider string `validate:"required,oneof=iyzico moka"`
	}{Provider: provider}); err != nil {
		return err
	}
	st := adaptershared.PaymentIntegrationStatus(p.Deps.Payment)
	if err := adaptershared.ValidatePaymentActivation(provider, st); err != nil {
		return err
	}
	if err := p.Deps.Settings.SetPaymentActiveProvider(ctx, provider); err != nil {
		return err
	}
	p.Notice = "Ödeme sağlayıcısı güncellendi"
	p.Redirect = "/dashboard/settings/payment"
	return c.Mount(ctx, p)
}

type paymentProviderCtrl struct {
	provider appsettings.PaymentProviderRow
}

func (c *paymentProviderCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	name := ""
	if p.Params != nil {
		name = p.Params["provider"]
	}
	st := adaptershared.PaymentIntegrationStatus(p.Deps.Payment)
	row, err := p.Deps.Settings.GetPaymentProvider(ctx, name, st)
	if err != nil {
		return err
	}
	c.provider = row
	return nil
}

func (c *paymentProviderCtrl) Render(p *Page) (string, error) {
	return p.RenderView("pages.payment_provider", map[string]any{
		"Name":        c.provider.Name,
		"Label":       c.provider.Label,
		"Description": c.provider.Description,
		"Active":      c.provider.Active,
		"Configured":  c.provider.Configured,
	})
}

func (c *paymentProviderCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermNotificationsSettings); err != nil {
		return err
	}
	if event != "payment.activate" {
		return nil
	}
	provider := payloadString(payload, "provider")
	name := ""
	if p.Params != nil {
		name = p.Params["provider"]
	}
	if err := validation.Check(p.Deps.Validate, &struct {
		Provider string `validate:"required,oneof=iyzico moka"`
	}{Provider: provider}); err != nil {
		return err
	}
	if provider != name {
		return domainsettings.ErrInvalidPaymentProvider
	}
	st := adaptershared.PaymentIntegrationStatus(p.Deps.Payment)
	if err := adaptershared.ValidatePaymentActivation(provider, st); err != nil {
		return err
	}
	if err := p.Deps.Settings.SetPaymentActiveProvider(ctx, provider); err != nil {
		return err
	}
	p.Notice = "Ödeme sağlayıcısı güncellendi"
	p.Redirect = "/dashboard/settings/payment"
	return c.Mount(ctx, p)
}
