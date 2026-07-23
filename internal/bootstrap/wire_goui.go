package bootstrap

import (
	"context"

	gouiweb "github.com/zatrano/gocore/internal/adapters/goui"
)

func (g *graph) wireGoUI() {
	ui, err := gouiweb.New(gouiweb.Deps{
		AuthDeps: gouiweb.AuthDeps{Auth: g.authServiceWeb},
		UserDeps: gouiweb.UserDeps{Users: g.userService},
		NotificationDeps: gouiweb.NotificationDeps{
			Notifications: g.notifService, InboxRealtime: g.rtHub,
		},
		ContactDeps: gouiweb.ContactDeps{Contacts: g.contactService},
		PaymentSettingsDeps: gouiweb.PaymentSettingsDeps{
			Settings: g.settingsSvc, Notify: g.cfg.Notify, Payment: g.cfg.Payment,
			ThreeDSSvc: g.paymentThreeDSSvc,
		},
		AuditDeps: gouiweb.AuditDeps{Audit: g.auditService},
		Authz:     g.authzService, Checker: g.authzResolver,
		Storage: g.localStorage, AllowedMIMEs: g.cfg.Security.AllowedUploadMIME,
		Upload:   g.uploadSvc,
		Validate: g.validate, Secure: g.cfg.App.IsProduction(), AccessTTL: g.cfg.Auth.AccessTokenTTL,
		MaxUpload: g.cfg.Security.MaxUploadBytes, Locales: g.cfg.I18n.Supported,
		Turnstile: g.turnstileClient, TurnstileSiteKey: g.cfg.Security.TurnstileSiteKey,
		Cache: g.memCache, Translator: g.translator,
		RateLimit: g.ipLimiter.Allow,
	})
	if err != nil {
		panic("goui: " + err.Error())
	}
	g.webUI = ui
	g.app.shutdownFns = append(g.app.shutdownFns, func(context.Context) error {
		g.rtHub.Close()
		g.webUI.Close()
		return nil
	})
}
