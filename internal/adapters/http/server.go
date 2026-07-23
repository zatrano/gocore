// Package http, Fiber tabanlı HTTP sunucusunu, middleware zincirini ve rota
// tanımlarını içerir. Bu katman "sürücü adaptörü"dür (hexagonal): dış dünyadan
// gelen istekleri uygulama use-case'lerine yönlendirir.
package http

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	gouiweb "github.com/zatrano/gocore/internal/adapters/goui"
	"github.com/zatrano/gocore/internal/adapters/http/handler"
	mw "github.com/zatrano/gocore/internal/adapters/http/middleware"
	"github.com/zatrano/gocore/internal/adapters/http/render"
	"github.com/zatrano/gocore/internal/application/auth"
	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/rbac"
)

// Deps, HTTP sunucusunun tüm bağımlılıklarıdır (constructor injection).
type Deps struct {
	Config     *config.Config
	Logger     *slog.Logger
	Sessions   *auth.SessionManager
	Authz      rbac.Checker
	Translator *i18n.Translator

	Health       *handler.HealthHandler
	User         *handler.UserHandler
	Auth         *handler.AuthHandler
	RBAC         *handler.RBACHandler
	Upload       *handler.UploadHandler
	Docs         *handler.DocsHandler
	Notification *handler.NotificationHandler
	Settings     *handler.SettingsHandler
	Payment      *handler.PaymentHandler
	Audit        *handler.AuditHandler
	Contact      *handler.ContactHandler
	Realtime     *handler.RealtimeHandler
	Web          *gouiweb.UI
	Cache        appshared.Cache
	Idempotency  *appidempotency.Service
}

// Server, Fiber uygulamasını ve yaşam döngüsünü sarmalar.
type Server struct {
	app *fiber.App
	cfg config.HTTP
	log *slog.Logger
}

// NewServer, Fiber uygulamasını yapılandırır, middleware'leri ve rotaları kurar.
func NewServer(d Deps) *Server {
	app := fiber.New(fiber.Config{
		AppName:      d.Config.App.Name,
		ServerHeader: "",
		BodyLimit:    d.Config.HTTP.BodyLimitBytes,
		ReadTimeout:  d.Config.HTTP.ReadTimeout,
		WriteTimeout: d.Config.HTTP.WriteTimeout,
		IdleTimeout:  d.Config.HTTP.IdleTimeout,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if isAPIRequest(c) {
				return render.Error(c, err)
			}
			if errors.Is(err, fiber.ErrNotFound) {
				return renderWebError(c, fiber.StatusNotFound, "Sayfa bulunamadı", "İstenen sayfa bulunamadı")
			}
			return renderWebError(c, fiber.StatusInternalServerError, "Hata", err.Error())
		},
	})

	registerMiddleware(app, d)
	registerRoutes(app, d)

	return &Server{app: app, cfg: d.Config.HTTP, log: d.Logger}
}

// registerMiddleware, global middleware zincirini kurar. Sıra önemlidir:
// recover en dışta (tüm panikleri yakalar), ardından güvenlik katmanları gelir.
func registerMiddleware(app *fiber.App, d Deps) {
	// Panic recovery — en dışta, her şeyi sarmalar.
	app.Use(recover.New(recover.Config{EnableStackTrace: !d.Config.App.IsProduction()}))

	// İstek kimliği + correlation + loglama.
	app.Use(requestid.New())
	app.Use(mw.Correlation())

	// Dil çözümleme (i18n) — hata/doğrulama mesajlarının yerelleştirilebilmesi
	// için loglama ve route'lardan önce çalışır.
	app.Use(mw.Locale(d.Translator))

	app.Use(mw.RequestLogger(d.Logger))

	// Güvenlik başlıkları (helmet): XSS, nosniff, frame-options, HSTS vb.
	app.Use(helmet.New(helmet.Config{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "no-referrer",
		CrossOriginEmbedderPolicy: "unsafe-none",
		HSTSMaxAge:                31536000,
		HSTSPreloadEnabled:        d.Config.App.IsProduction(),
	}))

	// CORS.
	app.Use(cors.New(cors.Config{
		AllowOrigins: splitCSV(d.Config.HTTP.CORSAllowOrigin),
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Accept-Language", "Authorization",
			mw.HeaderCorrelationID, mw.HeaderRequestID,
			"Idempotency-Key",
		},
		ExposeHeaders: []string{
			mw.HeaderCorrelationID, mw.HeaderRequestID,
		},
		AllowCredentials: false,
		MaxAge:           300,
	}))
}

// registerRoutes, tüm rotaları API versioning ile tanımlar.
func registerRoutes(app *fiber.App, d Deps) {
	// GoUI web arayüzü.
	if d.Web != nil {
		d.Web.Register(app)
	}

	// Probe'lar (versiyon dışı).
	app.Get("/livez", d.Health.Live)
	app.Get("/readyz", d.Health.Ready)
	app.Get("/healthz", d.Health.Health)

	// Genel canlı olay WebSocket (API / mobil / panel). Rate limiter dışında —
	// uzun ömürlü bağlantı; kimlik Authenticate içinde.
	if d.Realtime != nil {
		app.Get("/api/v1/ws", d.Realtime.Authenticate, d.Realtime.Upgrade())
	}

	// API dokümantasyonu.
	app.Get("/openapi.yaml", d.Docs.Spec)
	app.Get("/docs", d.Docs.SwaggerUI)

	// Rate limiter yalnızca API rotalarına uygulanır (IP başına throttling).
	apiLimiter := limiter.New(limiter.Config{
		Max:          d.Config.Security.RateLimitMax,
		Expiration:   d.Config.Security.RateLimitWindow,
		KeyGenerator: func(c fiber.Ctx) string { return c.IP() },
	})

	v1 := app.Group("/api/v1", apiLimiter)
	idem := mw.IdempotencyGuard(d.Idempotency)

	// 3DS callback / webhook — dış sağlayıcı; idempotency uygulanmaz (iş mantığı terminal state).
	v1.Post("/payments/3ds/callback", d.Payment.Callback3DS)
	v1.Post("/payments/webhook/iyzico", d.Payment.IyzicoWebhook)

	// Public iletişim formu (web ile parite).
	if d.Contact != nil {
		v1.Post("/contact", idem, d.Contact.Submit)
	}

	// Kimlik doğrulama (public mutasyonlar — idem IP ile).
	authGrp := v1.Group("/auth")
	authGrp.Post("/login", idem, d.Auth.Login)
	authGrp.Post("/refresh", idem, d.Auth.Refresh)
	authGrp.Post("/logout", idem, d.Auth.Logout)
	authGrp.Post("/forgot-password", idem, d.Auth.ForgotPassword)
	authGrp.Post("/reset-password", idem, d.Auth.ResetPassword)
	authGrp.Post("/verify-email", idem, d.Auth.VerifyEmail)
	authGrp.Post("/resend-verification", idem, d.Auth.ResendVerification)
	authGrp.Get("/oauth/:provider", d.Auth.OAuthStart)
	authGrp.Get("/oauth/:provider/callback", d.Auth.OAuthCallback)
	authProtected := authGrp.Group("", mw.Authenticator(d.Sessions), idem)
	authProtected.Post("/change-password", d.Auth.ChangePassword)
	authProtected.Post("/mfa/setup", d.Auth.MFASetup)
	authProtected.Post("/mfa/enable", d.Auth.MFAEnable)
	authProtected.Post("/mfa/disable", d.Auth.MFADisable)
	authProtected.Get("/permissions", d.Auth.Permissions)

	// Kullanıcı kaydı public; diğer kullanıcı işlemleri korumalı.
	users := v1.Group("/users")
	users.Post("/", idem, d.User.Register)

	protected := users.Group("", mw.Authenticator(d.Sessions), idem)
	protected.Get("/me", d.User.Me)
	protected.Patch("/me/name", d.User.ChangeMyName)
	protected.Patch("/me/email", d.User.ChangeMyEmail)
	protected.Patch("/me/phone", d.User.ChangeMyPhone)
	protected.Patch("/me/locale", d.User.ChangeLocale)
	protected.Get("/me/notifications", d.Notification.List)
	protected.Delete("/me/notifications", d.Notification.DeleteAll)
	protected.Get("/me/notifications/unread-count", d.Notification.UnreadCount)
	protected.Post("/me/notifications/read-all", d.Notification.MarkAllRead)
	protected.Post("/me/notifications/:id/read", d.Notification.MarkRead)
	protected.Delete("/me/notifications/:id", d.Notification.Delete)
	protected.Post("/create", mw.RequirePermission(d.Authz, rbac.PermUsersList), d.User.AdminCreate)
	protected.Get("/:id", d.User.Get)
	protected.Get("/", mw.RequirePermission(d.Authz, rbac.PermUsersList), d.User.List)
	protected.Patch("/:id/email", d.User.ChangeEmail)
	protected.Patch("/:id/phone", d.User.ChangePhone)
	protected.Patch("/:id/name", d.User.ChangeName)
	protected.Patch("/:id/role", mw.RequirePermission(d.Authz, rbac.PermUsersRoleChange), d.User.ChangeRole)
	protected.Post("/:id/activate", mw.RequirePermission(d.Authz, rbac.PermUsersActivate), d.User.Activate)
	protected.Delete("/:id", mw.RequirePermission(d.Authz, rbac.PermUsersDelete), d.User.Delete)
	protected.Post("/:id/restore", mw.RequirePermission(d.Authz, rbac.PermUsersRestore), d.User.Restore)

	notif := v1.Group("/notifications", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermNotificationsSend), idem)
	notif.Post("/send", d.Notification.Send)
	notif.Post("/bulk", d.Notification.BulkSend)
	notif.Post("/bulk/upload", d.Notification.BulkUpload)

	rbacGrp := v1.Group("/rbac", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermRBACManage), idem)
	rbacGrp.Get("/permissions", d.RBAC.ListPermissions)
	rbacGrp.Post("/permissions", d.RBAC.CreatePermission)
	rbacGrp.Patch("/permissions/:name", d.RBAC.UpdatePermission)
	rbacGrp.Get("/roles", d.RBAC.ListRoles)
	rbacGrp.Post("/roles", d.RBAC.CreateRole)
	rbacGrp.Get("/roles/:name", d.RBAC.GetRole)
	rbacGrp.Patch("/roles/:name", d.RBAC.UpdateRole)
	rbacGrp.Put("/roles/:name/permissions", d.RBAC.SetPermissions)
	rbacGrp.Delete("/roles/:name", d.RBAC.DeleteRole)

	settingsGrp := v1.Group("/settings", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermNotificationsSettings), idem)
	settingsGrp.Get("/sms", d.Settings.ListSMS)
	settingsGrp.Post("/sms", d.Settings.CreateSMS)
	settingsGrp.Get("/sms/:provider", d.Settings.GetSMSProvider)
	settingsGrp.Patch("/sms/:provider", d.Settings.UpdateSMSProviderByName)
	settingsGrp.Get("/payment", d.Settings.ListPayment)
	settingsGrp.Post("/payment", d.Settings.CreatePayment)
	settingsGrp.Get("/payment/:provider", d.Settings.GetPaymentProvider)
	settingsGrp.Patch("/payment/:provider", d.Settings.UpdatePaymentProvider)

	payments := v1.Group("/payments", mw.Authenticator(d.Sessions), idem)
	payments.Get("/transactions", mw.RequirePermission(d.Authz, rbac.PermPaymentsList), d.Payment.ListPayments)
	payments.Get("/transactions/:reference", mw.RequirePermission(d.Authz, rbac.PermPaymentsCharge, rbac.PermPaymentsList), d.Payment.GetPayment)
	paymentsCharge := payments.Group("", mw.RequirePermission(d.Authz, rbac.PermPaymentsCharge))
	paymentsCharge.Post("/bin-check", d.Payment.BinCheck)
	paymentsCharge.Post("/3ds/initialize", d.Payment.Initialize3DS)
	paymentsCharge.Post("/3ds/auth", d.Payment.Complete3DS)
	paymentsCharge.Post("/calc-amount", d.Payment.CalcPaymentAmount)

	auditGrp := v1.Group("/audit", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermAuditList))
	auditGrp.Get("/logs", d.Audit.ListLogs)
	auditGrp.Get("/logs/:id", d.Audit.GetLog)

	if d.Contact != nil {
		contactsGrp := v1.Group("/contacts", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermContactsList), idem)
		contactsGrp.Get("/", d.Contact.List)
		contactsGrp.Get("/:id", d.Contact.Get)
		contactsGrp.Post("/:id/read", d.Contact.MarkRead)
	}

	uploads := v1.Group("/uploads", mw.Authenticator(d.Sessions), mw.RequirePermission(d.Authz, rbac.PermUploadsCreate), idem)
	uploads.Post("/", d.Upload.Upload)
}

func isAPIRequest(c fiber.Ctx) bool {
	path := c.Path()
	return strings.HasPrefix(path, "/api/") || path == "/openapi.yaml" ||
		path == "/livez" || path == "/readyz" || path == "/healthz"
}

func renderWebError(c fiber.Ctx, status int, title, message string) error {
	c.Status(status)
	c.Type("html", "utf-8")
	return c.SendString(fmt.Sprintf(
		`<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><link rel="stylesheet" href="/goui/assets/app.css"></head><body><main class="gocore-main"><section class="card"><h1>%s</h1><p>%s</p><a href="/">Ana sayfa</a></section></main></body></html>`,
		html.EscapeString(title), html.EscapeString(title), html.EscapeString(message),
	))
}

// Start, sunucuyu dinlemeye başlar (bloklar).
func (s *Server) Start() error {
	s.log.Info("http sunucusu başlatılıyor", slog.String("addr", s.cfg.Addr()))
	return s.app.Listen(s.cfg.Addr())
}

// Shutdown, sunucuyu graceful biçimde kapatır (aktif istekleri bekler).
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http sunucusu kapatılıyor")
	return s.app.ShutdownWithContext(ctx)
}
