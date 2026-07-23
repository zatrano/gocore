package bootstrap

import (
	"context"
	"log/slog"
	"time"

	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	duser "github.com/zatrano/gocore/internal/domain/user"
	infraoutbox "github.com/zatrano/gocore/internal/infrastructure/outbox"
	"github.com/zatrano/gocore/pkg/worker"
)

func (g *graph) wireOutbox(_ context.Context) {
	sideEffects := map[string][]infraoutbox.SideEffect{
		duser.EventRegistered: {
			g.userNotifier.OnRegisteredPayload,
			func(ctx context.Context, p appoutbox.DomainEventPayload) error {
				id, err := duser.ParseID(p.AggregateID)
				if err != nil {
					return err
				}
				u, err := g.userRepo.FindByID(ctx, id)
				if err != nil {
					return err
				}
				return g.emailVerifier.Send(ctx, u)
			},
		},
		duser.EventActivated:    {g.userNotifier.OnActivatedPayload},
		duser.EventEmailChanged: {g.userNotifier.OnEmailChangedPayload},
	}
	outboxWorker := infraoutbox.NewWorker(g.outboxRepo, g.log)
	outboxWorker.Register(appoutbox.KindEmailSend, infraoutbox.EmailHandler(g.mailer))
	outboxWorker.Register(appoutbox.KindNotificationDispatch, infraoutbox.DispatchHandler(g.dispatcher))
	outboxWorker.Register(appoutbox.KindDomainEvent, infraoutbox.DomainEventHandler(g.auditor, g.log, sideEffects))
	go outboxWorker.Run(g.bgCtx) //nolint:contextcheck // kalıcı outbox worker

	go worker.Scheduler(g.bgCtx, 5*time.Minute, func(ctx context.Context) { //nolint:contextcheck // periyodik temizlik
		g.guard.Cleanup()
		g.memCache.Cleanup()
		g.tokenStore.Cleanup()
		_ = g.authTokenRepo.DeleteExpired(ctx)
	})

	go worker.Scheduler(g.bgCtx, 5*time.Minute, func(ctx context.Context) { //nolint:contextcheck // ödeme reconciliation
		if _, err := g.paymentThreeDSSvc.ReconcileStale(ctx, 5*time.Minute, 50); err != nil {
			g.log.Warn("payment reconciliation failed", slog.Any("error", err))
		}
	})
}
