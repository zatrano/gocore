package goui

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appauth "github.com/zatrano/gocore/internal/application/auth"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	appuser "github.com/zatrano/gocore/internal/application/user"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/pkg/rbac"
)

const (
	authExchangePrefix = "goui:auth:exchange:"
	authExchangeTTL    = 60 * time.Second
)

var (
	errAuthCacheUnavailable = errors.New("oturum önbelleği yapılandırılmamış")
	errAuthExchangeInvalid  = errors.New("oturum kodu geçersiz veya süresi dolmuş")
)

// registerUtilityRoutes, WS üzerinden HttpOnly cookie yazılamadığı için
// tek kullanımlık token exchange, çıkış, dil, OAuth ve ödeme callback
// uçlarını kaydeder.
func (ui *UI) registerUtilityRoutes(app *fiber.App) {
	app.Get("/goui/auth/exchange", ui.handleAuthExchange)
	app.Get("/goui/auth/clear", ui.handleAuthClear)
	app.Get("/lang/:locale", ui.handleSetLanguage)
	app.Get("/dashboard/payments/3ds/start", ui.handle3DSStart)
	app.Get("/dashboard/account/notifications/unread-count", ui.handleUnreadCount)
	ui.registerExportRoutes(app)

	if ui.deps.Auth != nil && ui.deps.Auth.OAuthHandler() != nil {
		app.Get("/auth/oauth/:provider", ui.handleOAuthStart)
		app.Get("/auth/oauth/:provider/callback", ui.handleOAuthCallback)
	}
}

func (ui *UI) handle3DSStart(c fiber.Ctx) error {
	claims, _, _, err := ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
	if err != nil {
		return c.Redirect().To("/auth/login?next=" + url.QueryEscape(c.OriginalURL()))
	}
	if !ui.anyPermission(c, claims.Role, []rbac.Permission{rbac.PermPaymentsCharge}) {
		return fiber.ErrForbidden
	}
	if ui.deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	result, err := ui.deps.ThreeDSSvc.Start3DSView(c.Context(), c.Query("reference"))
	if err != nil {
		return err
	}
	switch result.Provider {
	case domainsettings.ProviderMoka.String():
		if strings.TrimSpace(result.RedirectURL) == "" {
			return errors.New("3DS yönlendirme adresi bulunamadı")
		}
		return c.Redirect().To(result.RedirectURL)
	case domainsettings.ProviderIyzico.String():
		content, err := decodeIyzicoHTML(result.ThreeDSHtmlContent)
		if err != nil {
			return err
		}
		c.Type("html", "utf-8")
		return c.SendString(content)
	default:
		return domainsettings.ErrInvalidPaymentProvider
	}
}

func (ui *UI) handleUnreadCount(c fiber.Ctx) error {
	claims, _, _, err := ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if ui.deps.Notifications == nil {
		return c.JSON(fiber.Map{"count": 0})
	}
	n, err := ui.deps.Notifications.UnreadCount(c.Context(), claims.UserID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"count": n})
}

// authExchangeBlob, kısa ömürlü cache için opaque alan adları kullanır.
// TokenPair'i doğrudan marshal etmek G117 (secret serialization) üretir;
// bu blob yalnızca sunucu tarafı exchange cache'inde yaşar.
type authExchangeBlob struct {
	A string    `json:"a"`
	R string    `json:"r"`
	E time.Time `json:"e"`
	T string    `json:"t"`
}

// storeAuthExchange, token çiftini kısa TTL ile cache'e yazar ve tek kullanımlık kod döner.
func storeAuthExchange(ctx context.Context, cache appshared.Cache, tokens appauth.TokenPair) (string, error) {
	if cache == nil {
		return "", errAuthCacheUnavailable
	}
	code, err := randomToken(32)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(authExchangeBlob{
		A: tokens.AccessToken,
		R: tokens.RefreshToken,
		E: tokens.ExpiresAt,
		T: tokens.TokenType,
	})
	if err != nil {
		return "", err
	}
	if err := cache.Set(ctx, authExchangePrefix+code, raw, authExchangeTTL); err != nil {
		return "", err
	}
	return code, nil
}

// consumeAuthExchange, kodu tek seferlik tüketir ve token çiftini döner.
func consumeAuthExchange(ctx context.Context, cache appshared.Cache, code string) (appauth.TokenPair, error) {
	var zero appauth.TokenPair
	if cache == nil {
		return zero, errAuthCacheUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return zero, errAuthExchangeInvalid
	}
	key := authExchangePrefix + code
	raw, ok, err := cache.Take(ctx, key)
	if err != nil {
		return zero, err
	}
	if !ok || len(raw) == 0 {
		return zero, errAuthExchangeInvalid
	}
	var blob authExchangeBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
		return zero, errAuthExchangeInvalid
	}
	if blob.A == "" || blob.R == "" {
		return zero, errAuthExchangeInvalid
	}
	return appauth.TokenPair{
		AccessToken:  blob.A,
		RefreshToken: blob.R,
		ExpiresAt:    blob.E,
		TokenType:    blob.T,
	}, nil
}

func (ui *UI) handleAuthExchange(c fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Redirect().To("/auth/login")
	}
	tokens, err := consumeAuthExchange(c.Context(), ui.deps.Cache, code)
	if err != nil {
		return c.Redirect().To("/auth/login")
	}
	setAuthCookies(c, tokens, ui.deps.Secure, ui.deps.AccessTTL)
	next := loginRedirectURL(c.Query("next"))
	Flash(c, "success", "giriş başarılı")
	return c.Redirect().To(next)
}

func (ui *UI) handleAuthClear(c fiber.Ctx) error {
	clearAuthCookies(c, ui.deps.Secure)
	return c.Redirect().To("/auth/login")
}

func (ui *UI) handleSetLanguage(c fiber.Ctx) error {
	_, _, _, _ = ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
	locale := strings.ToLower(strings.TrimSpace(c.Params("locale")))
	supported := map[string]bool{}
	for _, loc := range ui.deps.Locales {
		supported[strings.ToLower(strings.TrimSpace(loc))] = true
	}
	if len(supported) == 0 {
		supported["tr"] = true
		supported["en"] = true
	}
	if !supported[locale] {
		return c.Redirect().To("/")
	}
	c.Cookie(&fiber.Cookie{
		Name:     adapters.LangCookie,
		Value:    locale,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		SameSite: fiber.CookieSameSiteLaxMode,
	})
	if userID, _ := adapters.ActorFromCtx(c); userID != "" && ui.deps.Users != nil {
		_, _ = ui.deps.Users.ChangeLocale(c.Context(), appuser.ChangeLocaleCommand{
			UserID: userID, Locale: locale,
		})
	}
	return c.Redirect().To(localizedBack(c, locale))
}

func localizedBack(c fiber.Ctx, locale string) string {
	if next := safeWebPath(c.Query("next")); next != "" {
		return localizedPath(next, locale)
	}
	ref := c.Get("Referer")
	if ref == "" {
		return "/"
	}
	u, err := url.Parse(ref)
	if err != nil || u.Path == "" {
		return "/"
	}
	back := u.Path
	if u.RawQuery != "" {
		back += "?" + u.RawQuery
	}
	return localizedPath(back, locale)
}

// localizedPath, dil tercihine göre yerelleştirilmiş public yolları eşler.
// TR iletişim: /iletisim · EN iletişim: /contact
func localizedPath(raw, locale string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "/"
	}
	switch u.Path {
	case "/contact", "/iletisim":
		u.Path = contactPath(locale)
	}
	return u.RequestURI()
}

func contactPath(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), "en") {
		return "/contact"
	}
	return "/iletisim"
}

func (ui *UI) handleOAuthStart(c fiber.Ctx) error {
	if access := c.Cookies(cookieAccess); access != "" && ui.deps.Auth != nil {
		if _, err := ui.deps.Auth.Verify(c.Context(), access); err == nil {
			return c.Redirect().To("/dashboard")
		}
	}
	provider := c.Params("provider")
	from := c.Query("from")
	if from != "register" {
		from = "login"
	}
	next := safeWebPath(c.Query("next"))
	state, err := adapters.IssueOAuthState(
		c.Context(),
		ui.deps.Cache,
		adapters.OAuthStateWebPrefix,
		adapters.OAuthStatePayload{Provider: provider, From: from, Next: next},
	)
	if err != nil {
		return c.Redirect().To(oauthAuthPath(from, next))
	}
	authURL, err := ui.deps.Auth.OAuthAuthCodeURL(provider, state)
	if err != nil {
		return c.Redirect().To(oauthAuthPath(from, next))
	}
	return c.Redirect().To(authURL)
}

func (ui *UI) handleOAuthCallback(c fiber.Ctx) error {
	provider := c.Params("provider")
	payload, ok := adapters.ConsumeOAuthState(
		c.Context(),
		ui.deps.Cache,
		adapters.OAuthStateWebPrefix,
		c.Query("state"),
		provider,
	)
	if !ok {
		return c.Redirect().To("/auth/login")
	}
	code := c.Query("code")
	if code == "" {
		return c.Redirect().To(oauthAuthPath(payload.From, payload.Next))
	}
	tokens, err := ui.deps.Auth.OAuthCallback(c.Context(), provider, code)
	if err != nil {
		return c.Redirect().To(oauthAuthPath(payload.From, payload.Next))
	}
	setAuthCookies(c, tokens, ui.deps.Secure, ui.deps.AccessTTL)
	next := safeWebPath(payload.Next)
	if next == "" {
		next = "/dashboard"
	}
	Flash(c, "success", "giriş başarılı")
	return c.Redirect().To(next)
}

func oauthAuthPath(from, next string) string {
	path := "/auth/login"
	if from == "register" {
		path = "/auth/register"
	}
	if next = safeWebPath(next); next != "" {
		path += "?next=" + url.QueryEscape(next)
	}
	return path
}

// redirectWithAuthExchange, WS login sonrası cookie yazımı için exchange URL'sine yönlendirir.
func redirectWithAuthExchange(ctx context.Context, p *Page, tokens appauth.TokenPair, next string) error {
	code, err := storeAuthExchange(ctx, p.Deps.Cache, tokens)
	if err != nil {
		if errors.Is(err, errAuthCacheUnavailable) {
			return errAuthCacheUnavailable
		}
		return err
	}
	dest := "/goui/auth/exchange?code=" + url.QueryEscape(code) +
		"&next=" + url.QueryEscape(loginRedirectURL(next))
	p.Redirect = dest
	// Toast, exchange sonrası cookie flash ile dashboard'da gösterilir.
	return nil
}

func loginRedirectURL(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || next[0] != '/' || (len(next) > 1 && next[1] == '/') {
		return "/dashboard"
	}
	u, err := url.Parse(next)
	if err != nil || isAuthPath(u.Path) {
		return "/dashboard"
	}
	return u.Path
}

func safeWebPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != '/' || strings.HasPrefix(raw, "//") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Path == "" {
		return ""
	}
	if u.RawQuery != "" {
		return u.Path + "?" + u.RawQuery
	}
	return u.Path
}

func isAuthPath(path string) bool {
	switch path {
	case "/auth/login", "/auth/register", "/auth/forgot-password", "/auth/reset-password",
		"/auth/verify-email":
		return true
	default:
		return false
	}
}
