package bootstrap

import (
	"context"
	"log/slog"

	"github.com/go-playground/validator/v10"

	gouiweb "github.com/zatrano/gocore/internal/adapters/goui"
	"github.com/zatrano/gocore/internal/adapters/http/handler"
	"github.com/zatrano/gocore/internal/adapters/persistence/postgres"
	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/application/authz"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/internal/infrastructure/cache"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/internal/infrastructure/email"
	"github.com/zatrano/gocore/internal/infrastructure/events"
	infranotif "github.com/zatrano/gocore/internal/infrastructure/notification"
	infrarealtime "github.com/zatrano/gocore/internal/infrastructure/realtime"
	"github.com/zatrano/gocore/internal/infrastructure/security"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	"github.com/zatrano/gocore/internal/infrastructure/storage"
	"github.com/zatrano/gocore/pkg/fieldenc"
	"github.com/zatrano/gocore/pkg/i18n"
)

// graph, Build sırasında oluşturulan ara bağımlılıkları taşır.
type graph struct {
	cfg *config.Config
	log *slog.Logger
	app *App

	translator *i18n.Translator

	txManager      *database.TxManager
	mfaFieldCipher *fieldenc.Cipher
	memCache       *cache.Memory

	bgCtx context.Context

	userRepo          *postgres.UserRepository
	mfaRepo           *postgres.MFARepository
	notifRepo         *postgres.NotificationRepository
	authTokenRepo     *postgres.AuthTokenRepository
	roleRepo          *postgres.RoleRepository
	outboxRepo        *postgres.OutboxRepository
	contactRepo       *postgres.ContactRepository
	publisher         *events.OutboxPublisher
	settingsSvc       *appsettings.Service
	idemSvc           *appidempotency.Service
	paymentThreeDSSvc *apppayment.ThreeDSService

	authzResolver *authz.Resolver
	authzService  *authz.Service
	roleChecker   *authz.RoleExistsChecker
	userAccess    appuser.Access

	hasher     *security.Argon2Hasher
	guard      *security.MemoryLoginGuard
	ipLimiter  *security.MemoryIPRateLimiter
	tokenStore *security.MemoryTokenStore
	sessions   *appauth.SessionManager
	totp       *security.TOTP

	mailer            appshared.Mailer
	outboxDispatch    *email.OutboxDispatcher
	notifListH        *appnotif.ListHandler
	notifMarkReadH    *appnotif.MarkReadHandler
	notifMarkAllReadH *appnotif.MarkAllReadHandler
	notifDeleteH      *appnotif.DeleteHandler
	notifDeleteAllH   *appnotif.DeleteAllHandler
	notifUnreadH      *appnotif.UnreadCountHandler
	rtHub             *infrarealtime.Hub
	dispatcher        *appnotif.Dispatcher
	asyncRunner       *infranotif.AsyncRunner
	userNotifier      *email.UserNotifier
	authNotifier      *email.AuthNotifier

	loginH         *appauth.LoginHandler
	changePwdH     *appauth.ChangePasswordHandler
	forgotH        *appauth.ForgotPasswordHandler
	resetH         *appauth.ResetPasswordHandler
	emailVerifier  *appauth.EmailVerifier
	mfaH           *appauth.MFAHandler
	oauthH         *appauth.OAuthHandler
	oauthWebH      *appauth.OAuthHandler
	authService    *appauth.Service
	authServiceWeb *appauth.Service

	auditor *postgres.AuditRepository

	localePolicy     appuser.LocalePolicy
	registerH        *appuser.RegisterHandler
	activateH        *appuser.ActivateHandler
	changeEmailH     *appuser.ChangeEmailHandler
	changePhoneH     *appuser.ChangePhoneHandler
	changeLocaleH    *appuser.ChangeLocaleHandler
	changeNameH      *appuser.ChangeNameHandler
	changeRoleH      *appuser.ChangeRoleHandler
	deleteH          *appuser.DeleteHandler
	restoreH         *appuser.RestoreHandler
	getH             *appuser.GetHandler
	listH            *appuser.ListHandler
	userService      *appuser.Service
	auditListH       *appaudit.ListHandler
	auditGetH        *appaudit.GetHandler
	auditService     *appaudit.Service
	contactSubmitH   *appcontact.SubmitHandler
	contactListH     *appcontact.ListHandler
	contactGetH      *appcontact.GetHandler
	contactMarkReadH *appcontact.MarkReadHandler
	contactService   *appcontact.Service
	manualSender     *appnotif.ManualSender
	notifService     *appnotif.Service

	validate            *validator.Validate
	turnstileClient     turnstile.Verifier
	localStorage        *storage.Local
	uploadSvc           *appupload.Service
	userHandler         *handler.UserHandler
	authHandler         *handler.AuthHandler
	rbacHandler         *handler.RBACHandler
	healthHandler       *handler.HealthHandler
	contactAPIHandler   *handler.ContactHandler
	uploadHandler       *handler.UploadHandler
	docsHandler         *handler.DocsHandler
	notificationHandler *handler.NotificationHandler
	settingsHandler     *handler.SettingsHandler
	paymentHandler      *handler.PaymentHandler
	auditAPIHandler     *handler.AuditHandler
	realtimeHandler     *handler.RealtimeHandler

	webUI *gouiweb.UI
}
