package bootstrap

import (
	httpadapter "github.com/zatrano/gocore/internal/adapters/http"
)

func (g *graph) wireServer() {
	g.app.Server = httpadapter.NewServer(httpadapter.Deps{ //nolint:contextcheck // fiber middleware kendi context'ini yönetir
		Config:       g.cfg,
		Logger:       g.log,
		Sessions:     g.sessions,
		Authz:        g.authzResolver,
		Translator:   g.translator,
		Health:       g.healthHandler,
		User:         g.userHandler,
		Auth:         g.authHandler,
		RBAC:         g.rbacHandler,
		Upload:       g.uploadHandler,
		Docs:         g.docsHandler,
		Notification: g.notificationHandler,
		Settings:     g.settingsHandler,
		Payment:      g.paymentHandler,
		Audit:        g.auditAPIHandler,
		Contact:      g.contactAPIHandler,
		Realtime:     g.realtimeHandler,
		Web:          g.webUI,
		Cache:        g.memCache,
		Idempotency:  g.idemSvc,
	})
}
