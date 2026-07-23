package bootstrap

import (
	"github.com/go-playground/validator/v10"

	"github.com/zatrano/gocore/internal/adapters/http/handler"
	appupload "github.com/zatrano/gocore/internal/application/upload"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/pkg/validation"
)

func (g *graph) wireHTTP() error {
	g.validate = validator.New(validator.WithRequiredStructEnabled())
	validation.Register(g.validate)
	g.turnstileClient = turnstile.NewClient(g.cfg.Security.TurnstileSiteKey, g.cfg.Security.TurnstileSecretKey.Value())
	g.userHandler = handler.NewUserHandler(handler.UserDeps{
		Users: g.userService, Validate: g.validate, Turnstile: g.turnstileClient,
	})
	g.authHandler = handler.NewAuthHandler(handler.AuthDeps{
		Auth: g.authService, Checker: g.authzResolver, Validate: g.validate,
		Secure: g.cfg.App.IsProduction(), Turnstile: g.turnstileClient, Cache: g.memCache,
	})
	g.rbacHandler = handler.NewRBACHandler(g.authzService, g.validate)
	g.healthHandler = handler.NewHealthHandler(g.app.pool, g.cfg.App.Version)
	g.contactAPIHandler = handler.NewContactHandler(handler.ContactDeps{
		Contacts: g.contactService, Validate: g.validate, Turnstile: g.turnstileClient,
	})

	localStorage, err := storageOrTemp(g.cfg)
	if err != nil {
		return err
	}
	g.localStorage = localStorage
	g.uploadSvc = appupload.NewService(g.localStorage, g.cfg.Security.MaxUploadBytes, g.cfg.Security.AllowedUploadMIME, g.publisher)
	g.uploadHandler = handler.NewUploadHandler(g.uploadSvc)
	g.docsHandler = handler.NewDocsHandler()
	g.notificationHandler = handler.NewNotificationHandler(handler.NotificationDeps{
		Notifications: g.notifService, Realtime: g.rtHub,
		Validate: g.validate, MaxBytes: g.cfg.Security.MaxUploadBytes,
	})
	g.settingsHandler = handler.NewSettingsHandler(g.settingsSvc, g.cfg.Notify, g.cfg.Payment, g.validate)
	g.paymentHandler = handler.NewPaymentHandler(g.paymentThreeDSSvc, g.validate)
	g.auditAPIHandler = handler.NewAuditHandler(g.auditService)
	g.realtimeHandler = handler.NewRealtimeHandler(g.rtHub, g.sessions, g.log)
	return nil
}
