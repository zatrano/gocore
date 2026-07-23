package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres"
	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	"github.com/zatrano/gocore/internal/infrastructure/events"
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
	"github.com/zatrano/gocore/internal/infrastructure/payment/moka"
	"github.com/zatrano/gocore/pkg/fieldenc"
)

func (g *graph) wireRepos(ctx context.Context) error {
	g.userRepo = postgres.NewUserRepository(g.txManager, g.mfaFieldCipher)
	g.mfaRepo = postgres.NewMFARepository(g.txManager)
	if err := g.userRepo.EncryptLegacyMFASecrets(ctx); err != nil {
		return fmt.Errorf("bootstrap: eski MFA secret şifreleme: %w", err)
	}
	g.notifRepo = postgres.NewNotificationRepository(g.txManager)
	g.authTokenRepo = postgres.NewAuthTokenRepository(g.txManager)
	g.roleRepo = postgres.NewRoleRepository(g.txManager)
	settingsRepo := postgres.NewSettingsRepository(g.txManager)
	g.outboxRepo = postgres.NewOutboxRepository(g.txManager)
	g.contactRepo = postgres.NewContactRepository(g.txManager)
	g.auditor = postgres.NewAuditRepository(g.txManager)

	g.publisher = events.NewOutboxPublisher(g.outboxRepo)
	g.settingsSvc = appsettings.NewService(appsettings.SettingsDeps{Repo: settingsRepo, Publisher: g.publisher, Tx: g.txManager})

	paymentFieldCipher, err := fieldenc.New(g.cfg.Payment.FieldEncryptionKey.Value())
	if err != nil {
		return fmt.Errorf("bootstrap: ödeme alan şifreleme: %w", err)
	}
	paymentThreeDSRepo := postgres.NewPaymentRepository(g.txManager, paymentFieldCipher)
	idemRepo := postgres.NewIdempotencyRepository(g.txManager)
	g.idemSvc = appidempotency.NewService(idemRepo, 24*time.Hour)
	iyzicoClient := iyzico.NewClient(g.cfg.Payment)
	mokaClient := moka.NewClient(g.cfg.Payment)
	g.paymentThreeDSSvc = apppayment.NewThreeDSService(apppayment.ThreeDSDeps{
		Settings: g.settingsSvc, Repo: paymentThreeDSRepo, Iyzico: iyzicoClient, Moka: mokaClient,
		PaymentCfg: g.cfg.Payment, BaseURL: g.cfg.Auth.EmailLinkBaseURL, Idem: g.idemSvc,
		Publisher: g.publisher, Tx: g.txManager,
	})
	return nil
}
