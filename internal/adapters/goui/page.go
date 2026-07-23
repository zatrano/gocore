package goui

import (
	"context"
	"crypto/subtle"
	"errors"
	"html"
	htmltemplate "html/template"
	"regexp"
	"strconv"
	"strings"

	"github.com/zatrano/goui/core"
	gouitemplate "github.com/zatrano/goui/template"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	pkgi18n "github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/rbac"
)

var errUnauthenticated = errors.New("oturum geçersiz veya süresi dolmuş")
var submitFormPattern = regexp.MustCompile(`(?i)(<form\b[^>]*\bg-submit="[^"]+"[^>]*>)`)

// Controller contains one page's stateful GoUI behavior.
type Controller interface {
	Mount(context.Context, *Page) error
	Render(*Page) (string, error)
	HandleEvent(context.Context, *Page, string, map[string]any) error
}

// Page is the common GoUI lifecycle host used by every browser screen.
type Page struct {
	core.BaseComponent

	Deps          Deps
	Views         *gouitemplate.Registry
	Controller    Controller
	Screen        string
	Title         string
	Section       string
	Protected     bool
	RequiredPerms []rbac.Permission
	AccessToken   string
	RefreshToken  string
	Actor         appauth.Claims
	ClientIP      string
	UserAgent     string
	CorrelationID string
	Params        map[string]string
	Query         map[string]string
	Notice        string
	NoticeKind    string // success | info | warning — SweetAlert türü
	Error         string
	Redirect      string
	EventNonce    string
	UnreadCount   int64 // panel header bildirim rozeti
}

func (p *Page) Mount(ctx context.Context) error {
	ctx, err := p.actorContext(ctx)
	if err != nil {
		p.Error = err.Error()
		p.Redirect = "/auth/login"
		return nil //nolint:nilerr // oturum yok: sayfa login'e yönlendirilir
	}
	if err := p.ensureRequiredPerms(ctx); err != nil {
		p.Error = err.Error()
		p.MarkDirty()
		return nil //nolint:nilerr // yetkisiz: kullanıcıya sayfa uyarısı
	}
	if p.Controller == nil {
		return errors.New("goui: page controller is required")
	}
	if p.EventNonce == "" {
		p.EventNonce, err = randomToken(24)
		if err != nil {
			return err
		}
	}
	if err := p.Controller.Mount(ctx, p); err != nil {
		p.Error = err.Error()
	}
	p.refreshUnreadCount(ctx)
	p.MarkDirty()
	return nil
}

func (p *Page) Render() (string, error) {
	body, err := p.Controller.Render(p)
	if err != nil {
		return "", err
	}
	body = submitFormPattern.ReplaceAllString(body, `${1}<input type="hidden" name="_goui_nonce" value="`+html.EscapeString(p.EventNonce)+`">`)
	if p.Protected {
		body = p.translatePanelHTML(body)
	}
	htmlOut, err := p.RenderView("layouts.shell", p.shellData(body))
	if err != nil {
		return "", err
	}
	p.ResetDirty()
	return htmlOut, nil
}

// Head, ModeSEO / ModeStatic belge meta verisini sağlar (core.HeadProvider).
func (p *Page) Head() core.Head {
	h := core.Head{
		Title: p.Title,
		Lang:  p.Locale,
	}
	switch p.Screen {
	case "home":
		h.Description = p.T("public.home.meta_description",
			"Kullanıcı, yetkilendirme ve bildirim yönetimini tek yerden yönetin.")
		h.OGTitle = p.Title
		h.OGDescription = h.Description
	case "contact":
		h.Description = p.T("public.contact.meta_description",
			"Sorularınız için iletişim formunu doldurun.")
		h.OGTitle = p.Title
		h.OGDescription = h.Description
	}
	return h
}

// RenderView, Blade tarzı .goui.html şablonunu derlenmiş registry üzerinden üretir.
func (p *Page) RenderView(name string, data any) (string, error) {
	views := p.Views
	if views == nil {
		var err error
		views, err = testViews()
		if err != nil {
			return "", err
		}
	}
	return views.Render(name, data)
}

func (p *Page) shellData(body string) ShellData {
	authShell := !p.Protected && isGuestScreen(p.Screen)
	notice := p.Notice
	if notice != "" && p.Protected {
		notice = p.translatePanelNotice(notice)
	}
	data := ShellData{
		Protected:   p.Protected,
		AuthShell:   authShell,
		LoggedIn:    p.Actor.UserID != "",
		Title:       p.Title,
		Redirect:    p.Redirect,
		Notice:      notice,
		NoticeKind:  p.NoticeKind,
		Error:       p.Error,
		Body:        htmltemplate.HTML(body), // #nosec G203 -- controller + nonce enjeksiyonu sunucu üretir
		Locales:     p.shellLocales(),
		ContactPath: contactPath(p.Locale),
		Labels: ShellLabels{
			OpenMenu:      p.T("common.open_menu", "Menüyü aç"),
			CloseMenu:     p.T("common.close_menu", "Menüyü kapat"),
			Logout:        p.T("common.logout", "Çıkış"),
			Contact:       p.T("public.nav.contact", "İletişim"),
			Dashboard:     p.T("public.nav.dashboard", "Panel"),
			Login:         p.T("public.nav.login", "Giriş"),
			Register:      p.T("public.nav.register", "Kayıt"),
			Notifications: p.T("dashboard.nav.notifications", "Bildirimler"),
		},
		ActorRole:   p.Actor.Role,
		ActorEmail:  p.Actor.Email,
		ActorUserID: p.Actor.UserID,
		UnreadCount: p.UnreadCount,
	}
	if p.Protected {
		data.NavItems = p.shellNavItems()
		data.UnreadBadge = "0"
		data.NotifAriaLabel = data.Labels.Notifications
		if p.UnreadCount > 0 {
			data.HasUnread = true
			if p.UnreadCount > 99 {
				data.UnreadBadge = "99+"
			} else {
				data.UnreadBadge = strconv.FormatInt(p.UnreadCount, 10)
			}
			data.NotifAriaLabel = data.Labels.Notifications + " (" + data.UnreadBadge + ")"
		}
	}
	return data
}

func (p *Page) shellLocales() []ShellLocale {
	locales := p.Deps.Locales
	if len(locales) == 0 {
		locales = []string{"tr", "en"}
	}
	current := strings.ToLower(strings.TrimSpace(p.Locale))
	out := make([]ShellLocale, 0, len(locales))
	for _, locale := range locales {
		locale = strings.ToLower(strings.TrimSpace(locale))
		if locale == "" {
			continue
		}
		out = append(out, ShellLocale{
			Code:     locale,
			Label:    localeLabel(locale),
			Selected: locale == current,
		})
	}
	return out
}

func (p *Page) shellNavItems() []ShellNavItem {
	items := []ShellNavItem{
		{Href: "/dashboard", Icon: "dashboard", Label: p.T("dashboard.nav.dashboard", "Dashboard"), Active: p.Section == "dashboard"},
	}
	if p.Allowed(context.Background(), rbac.PermUsersList) {
		items = append(items, ShellNavItem{Href: "/dashboard/users", Icon: "users", Label: p.T("dashboard.nav.users", "Kullanıcılar"), Active: p.Section == "users"})
	}
	if p.Allowed(context.Background(), rbac.PermContactsList) {
		items = append(items, ShellNavItem{Href: "/dashboard/contacts", Icon: "mail", Label: p.T("dashboard.nav.contacts", "İletişim"), Active: p.Section == "contacts"})
	}
	if p.Allowed(context.Background(), rbac.PermRBACManage) {
		items = append(items, ShellNavItem{Href: "/dashboard/rbac/roles", Icon: "shield", Label: p.T("dashboard.nav.rbac", "RBAC"), Active: p.Section == "rbac"})
	}
	if p.Allowed(context.Background(), rbac.PermNotificationsSend) {
		items = append(items, ShellNavItem{Href: "/dashboard/notifications/send", Icon: "bell", Label: p.T("dashboard.nav.notifications", "Bildirimler"), Active: p.Section == "notifications"})
	}
	if p.Allowed(context.Background(), rbac.PermPaymentsCharge) {
		items = append(items, ShellNavItem{Href: "/dashboard/payments/checkout", Icon: "card", Label: p.T("dashboard.nav.payment", "Ödeme"), Active: p.Section == "payments-checkout"})
	}
	if p.Allowed(context.Background(), rbac.PermPaymentsList) {
		items = append(items, ShellNavItem{Href: "/dashboard/payments/transactions", Icon: "receipt", Label: p.T("dashboard.nav.payments", "Ödemeler"), Active: p.Section == "payments-list"})
	}
	if p.Allowed(context.Background(), rbac.PermAuditList) {
		items = append(items, ShellNavItem{Href: "/dashboard/audit/logs", Icon: "audit", Label: p.T("dashboard.nav.audit", "Denetim"), Active: p.Section == "audit"})
	}
	if p.Allowed(context.Background(), rbac.PermNotificationsSettings) {
		items = append(items, ShellNavItem{
			Group: true,
			Open:  strings.HasPrefix(p.Section, "settings"),
			Icon:  "settings",
			Label: p.T("dashboard.nav.settings", "Ayarlar"),
			Children: []ShellNavItem{
				{Href: "/dashboard/settings/sms", Icon: "mail", Label: p.T("dashboard.settings.sms.title", "SMS Ayarları"), Active: p.Section == "settings-sms"},
				{Href: "/dashboard/settings/payment", Icon: "card", Label: p.T("dashboard.settings.payments.title", "Ödeme Ayarları"), Active: p.Section == "settings-payment"},
			},
		})
	}
	if p.Allowed(context.Background(), rbac.PermUploadsCreate) {
		items = append(items, ShellNavItem{Href: "/dashboard/uploads", Icon: "folder", Label: p.T("dashboard.nav.uploads", "Dosyalar"), Active: p.Section == "uploads"})
	}
	items = append(items, ShellNavItem{Href: "/dashboard/account", Icon: "account", Label: p.T("dashboard.nav.account", "Hesabım"), Active: p.Section == "account"})
	return items
}

func (p *Page) HandleEvent(ctx context.Context, event string, payload map[string]any) error {
	ctx, err := p.actorContext(ctx)
	if err != nil {
		p.Error = err.Error()
		p.Redirect = "/auth/login"
		p.MarkDirty()
		return nil //nolint:nilerr // oturum yok: sayfa login'e yönlendirilir
	}
	p.Error = ""
	p.Notice = ""
	p.NoticeKind = ""
	if err := p.ensureRequiredPerms(ctx); err != nil {
		p.Error = err.Error()
		p.MarkDirty()
		return nil //nolint:nilerr
	}
	if err := p.consumeRateLimit(event); err != nil {
		p.Error = err.Error()
		p.MarkDirty()
		return nil //nolint:nilerr
	}
	if fields, submitted := payload["fields"].(map[string]any); submitted {
		nonce, _ := fields["_goui_nonce"].(string)
		if nonce == "" || subtle.ConstantTimeCompare([]byte(nonce), []byte(p.EventNonce)) != 1 {
			p.Error = "Geçersiz veya daha önce kullanılmış form olayı."
			p.MarkDirty()
			return nil
		}
		nextNonce, nonceErr := randomToken(24)
		if nonceErr != nil {
			return nonceErr
		}
		p.EventNonce = nextNonce
	}
	if event == "session.logout" {
		if p.Deps.Auth != nil {
			if err := p.Deps.Auth.Logout(ctx, p.AccessToken, p.RefreshToken); err != nil {
				p.Error = err.Error()
				p.MarkDirty()
				return nil //nolint:nilerr // çıkış hatası kullanıcıya sayfa uyarısı olarak gösterilir
			}
		}
		p.Redirect = "/goui/auth/clear"
		p.MarkDirty()
		return nil
	}
	if err := p.Controller.HandleEvent(ctx, p, event, payload); err != nil {
		p.Error = err.Error()
	}
	p.refreshUnreadCount(ctx)
	p.MarkDirty()
	return nil
}

func (p *Page) Unmount(context.Context) error { return nil }

func (p *Page) refreshUnreadCount(ctx context.Context) {
	if !p.Protected || p.Deps.Notifications == nil || p.Actor.UserID == "" {
		p.UnreadCount = 0
		return
	}
	n, err := p.Deps.Notifications.UnreadCount(ctx, p.Actor.UserID)
	if err != nil {
		return
	}
	p.UnreadCount = n
}

func (p *Page) actorContext(ctx context.Context) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p.Deps.Translator != nil {
		ctx = pkgi18n.NewContext(ctx, p.Deps.Translator, pkgi18n.Locale(p.Locale))
	}
	if !p.Protected {
		return appshared.WithActor(ctx, appshared.ActorContext{
			ActorType:     appshared.ActorTypeAnonymous,
			Source:        appshared.SourceWeb,
			IP:            p.ClientIP,
			UserAgent:     p.UserAgent,
			CorrelationID: p.CorrelationID,
		}), nil
	}
	if p.Deps.Auth == nil || p.AccessToken == "" {
		return ctx, errUnauthenticated
	}
	claims, err := p.Deps.Auth.Verify(ctx, p.AccessToken)
	if err != nil {
		return ctx, errUnauthenticated
	}
	p.Actor = claims
	return appshared.WithActor(ctx, appshared.ActorContext{
		ActorID:       claims.UserID,
		ActorType:     appshared.ActorTypeUser,
		ActorEmail:    claims.Email,
		Source:        appshared.SourceWeb,
		IP:            p.ClientIP,
		UserAgent:     p.UserAgent,
		CorrelationID: p.CorrelationID,
	}), nil
}

func (p *Page) Allowed(ctx context.Context, permission rbac.Permission) bool {
	if !p.Protected || p.Deps.Checker == nil {
		return false
	}
	ok, err := p.Deps.Checker.Allows(ctx, p.Actor.Role, permission)
	return err == nil && ok
}

func (p *Page) ensureRequiredPerms(ctx context.Context) error {
	if len(p.RequiredPerms) == 0 {
		return nil
	}
	for _, perm := range p.RequiredPerms {
		if p.Allowed(ctx, perm) {
			return nil
		}
	}
	return errors.New("bu işlem için yetkiniz yok")
}

func (p *Page) consumeRateLimit(event string) error {
	if p == nil || p.Deps.RateLimit == nil || !isRateLimitedEvent(event) {
		return nil
	}
	key := strings.TrimSpace(p.ClientIP)
	if key == "" {
		key = "unknown"
	}
	if p.Deps.RateLimit(key) {
		return nil
	}
	return errors.New("çok fazla istek; lütfen kısa süre sonra tekrar deneyin")
}

func isRateLimitedEvent(event string) bool {
	switch event {
	case "login.submit", "register.submit", "forgot.submit",
		"reset.submit", "verify.resend", "contact.submit":
		return true
	default:
		return false
	}
}

// T, sayfa diline göre pkg/i18n çevirisi döner.
func (p *Page) T(key, fallback string, args ...any) string {
	if p == nil || p.Deps.Translator == nil {
		return fallback
	}
	loc := pkgi18n.Locale(strings.ToLower(strings.TrimSpace(p.Locale)))
	if loc == "" {
		loc = p.Deps.Translator.DefaultLocale()
	}
	return p.Deps.Translator.T(loc, key, fallback, args...)
}

func localeLabel(locale string) string {
	switch locale {
	case "tr":
		return "Türkçe"
	case "en":
		return "English"
	default:
		return strings.ToUpper(locale)
	}
}

func navIcon(name string) string {
	icons := map[string]string{
		"account":   `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 21a8 8 0 0 0-16 0M12 13a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z"/></svg>`,
		"audit":     `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 5h6M9 9h6M9 13h4M5 3h14v18H5z"/></svg>`,
		"bell":      `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/></svg>`,
		"card":      `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18v12H3zM3 10h18M7 15h3"/></svg>`,
		"close":     `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 6 12 12M18 6 6 18"/></svg>`,
		"dashboard": `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 4h6v6H4zM14 4h6v6h-6zM4 14h6v6H4zM14 14h6v6h-6z"/></svg>`,
		"folder":    `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h7l2 2h9v11H3z"/></svg>`,
		"mail":      `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16v12H4zM4 6l8 7 8-7"/></svg>`,
		"menu":      `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h16"/></svg>`,
		"receipt":   `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 3h12v18l-3-2-3 2-3-2-3 2zM9 8h6M9 12h6"/></svg>`,
		"settings":  `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7ZM19.4 15l1.1 2-2.5 2.5-2-1.1-2.2.9-.7 2.2H9.5l-.7-2.2-2.2-.9-2 1.1L2 17l1.1-2-.9-2.2L0 12V9l2.2-.8.9-2.2L2 4l2.6-2.5 2 1.1 2.2-.9L9.5 0h3l.7 1.7 2.2.9 2-1.1L20 4l-1.1 2 .9 2.2L22 9v3l-2.2.8z"/></svg>`,
		"chevron":   `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6"/></svg>`,
		"shield":    `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 20 6v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6zM9 12l2 2 4-4"/></svg>`,
		"users":     `<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM22 21v-2a4 4 0 0 0-3-3.9M16 3.1a4 4 0 0 1 0 7.8"/></svg>`,
	}
	return icons[name]
}

func payloadFields(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if fields, ok := payload["fields"].(map[string]any); ok {
		return fields
	}
	return payload
}

func payloadString(payload map[string]any, key string) string {
	v, ok := payloadFields(payload)[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func payloadStrings(payload map[string]any, key string) []string {
	v := payloadFields(payload)[key]
	switch value := v.(type) {
	case string:
		if value == "" {
			return nil
		}
		return []string{value}
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
