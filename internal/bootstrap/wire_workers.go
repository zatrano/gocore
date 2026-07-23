package bootstrap

import (
	"context"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	"github.com/zatrano/gocore/internal/infrastructure/email"
	infranotif "github.com/zatrano/gocore/internal/infrastructure/notification"
	"github.com/zatrano/gocore/internal/infrastructure/notification/sms"
	infrarealtime "github.com/zatrano/gocore/internal/infrastructure/realtime"
	"github.com/zatrano/gocore/pkg/worker"
)

func (g *graph) wireWorkers() {
	bgCtx, cancelBg := context.WithCancel(context.Background())
	g.bgCtx = bgCtx
	g.app.cancelBg = cancelBg
	g.app.workers = worker.NewPool(bgCtx, 4, 256, g.log) //nolint:contextcheck // arka plan worker havuzu

	g.mailer = email.NewMailer(g.cfg.SMTP, g.log)
	g.outboxDispatch = email.NewOutboxDispatcher(g.outboxRepo)

	smsRegistry := sms.BuildRegistry(g.cfg.Notify, g.settingsSvc, g.log)
	g.notifListH = appnotif.NewListHandler(g.notifRepo)
	g.notifMarkReadH = appnotif.NewMarkReadHandler(g.notifRepo)
	g.notifMarkAllReadH = appnotif.NewMarkAllReadHandler(g.notifRepo)
	g.notifDeleteH = appnotif.NewDeleteHandler(g.notifRepo)
	g.notifDeleteAllH = appnotif.NewDeleteAllHandler(g.notifRepo)
	g.notifUnreadH = appnotif.NewUnreadCountHandler(g.notifRepo)
	g.rtHub = infrarealtime.NewHub(g.notifUnreadH.Handle)
	inAppChannel := infranotif.NewInAppChannel(g.notifRepo, g.rtHub)
	g.dispatcher = appnotif.NewDispatcher(
		i18nTranslator{tr: g.translator},
		g.cfg.I18n.DefaultLocale,
		infranotif.NewEmailChannel(g.mailer),
		infranotif.NewSMSChannel(smsRegistry),
		inAppChannel,
	)
	g.asyncRunner = infranotif.NewAsyncRunner(g.app.workers, g.log)
	g.userNotifier = email.NewUserNotifier(g.outboxDispatch, g.userRepo)
	g.authNotifier = email.NewAuthNotifier(g.outboxDispatch, g.cfg.Auth.EmailLinkBaseURL)
}
