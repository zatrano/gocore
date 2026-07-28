package goui

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	gouifiber "github.com/zatrano/goui/adapters/fiber"
	"github.com/zatrano/goui/core"
	gouii18n "github.com/zatrano/goui/i18n"
	gouitemplate "github.com/zatrano/goui/template"
	gouiupload "github.com/zatrano/goui/upload"
	"github.com/zatrano/goui/ws"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
	pkgi18n "github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/rbac"
)

const (
	cookieAccess  = "access_token"
	cookieRefresh = "refresh_token"
	cookieCSRF    = "csrf_token"
)

type routeSpec struct {
	path       string
	screen     string
	title      string
	titleKey   string
	section    string
	protected  bool
	permission []rbac.Permission
	// mode, GoUI v1 page delivery: ModeLive (panel) veya ModeSEO (public SSR+hydrate).
	mode core.PageMode
}

// UI owns the GoUI registry, WebSocket hub and Fiber route integration.
type UI struct {
	deps         Deps
	registry     *core.Registry
	hub          *ws.Hub
	translator   *gouii18n.Translator
	server       *ws.Server
	store        gouiupload.Storage
	views        *gouitemplate.Registry
	viewsCleanup func()
	bindings     sync.Map
}

type componentBinding struct {
	protected  bool
	tokenID    string
	permission []rbac.Permission
}

func New(deps Deps) (*UI, error) {
	registry := core.NewRegistry()
	hub := ws.NewHub()
	translator := gouii18n.NewTranslator()
	views, cleanup, err := openViews(deps, hub)
	if err != nil {
		hub.Stop()
		return nil, err
	}
	return &UI{
		deps:         deps,
		registry:     registry,
		hub:          hub,
		translator:   translator,
		server:       ws.NewServer(hub, registry, translator),
		store:        NewUploadStore(deps.Storage, deps.MaxUpload, deps.AllowedMIMEs),
		views:        views,
		viewsCleanup: cleanup,
	}, nil
}

func (ui *UI) Close() {
	ui.hub.Stop()
	if ui.views != nil {
		ui.views.Close()
	}
	if ui.viewsCleanup != nil {
		ui.viewsCleanup()
	}
}

func (ui *UI) Register(app *fiber.App) {
	registerAssetRoutes(app)
	ui.registerUtilityRoutes(app)

	for _, spec := range pageRoutes() {
		spec := spec
		app.Get(spec.path, ui.pageHandler(spec))
	}

	// Geçici upload deposu: API ile aynı şekilde uploads:create gerekir.
	// Dosyadan bildirim gönderen rollerin de bu izne sahip olması gerekir.
	app.Use(gouiupload.UploadPath, ui.requireUploadAccess)
	app.Use(gouiupload.FilesPrefix, ui.requireUploadAccess)
	app.Use(ws.Path, requireSameOrigin)
	app.Use(ws.Path, ui.authorizeWebSocket)
	gouifiber.Register(app, gouifiber.Options{Server: ui.server, Store: ui.store})
}

func (ui *UI) pageHandler(spec routeSpec) fiber.Handler {
	return func(c fiber.Ctx) error {
		csrf := ensureCSRFCookie(c, ui.deps.Secure)
		var (
			claims  appauth.Claims
			access  = c.Cookies(cookieAccess)
			refresh = c.Cookies(cookieRefresh)
		)
		if spec.protected {
			var err error
			claims, access, refresh, err = ui.authenticate(c, access, refresh)
			if err != nil {
				next := c.Path()
				if raw := string(c.Request().URI().QueryString()); raw != "" {
					next += "?" + raw
				}
				return c.Redirect().To("/auth/login?next=" + url.QueryEscape(next))
			}
			if len(spec.permission) > 0 && !ui.anyPermission(c, claims.Role, spec.permission) {
				return c.Status(fiber.StatusForbidden).SendString("Bu işlem için yetkiniz yok.")
			}
		} else if access != "" {
			if current, err := ui.deps.Auth.Verify(c.Context(), access); err == nil {
				claims = current
				if isGuestScreen(spec.screen) {
					return c.Redirect().To("/dashboard")
				}
			}
		}

		params := routeParams(c)
		query := routeQuery(c)
		clientIP := c.IP()
		userAgent := c.Get("User-Agent")
		correlationID := c.Get("X-Correlation-ID")
		requestKey, err := randomToken(12)
		if err != nil {
			return err
		}
		name := componentName(spec, claims, csrf+"."+requestKey, params, query)
		locale := ui.localeFromRequest(c)
		title := ui.routeTitle(locale, spec)
		flashKind, flashMsg := ConsumeFlash(c)
		var flashMu sync.Mutex
		pendingKind, pendingMsg := flashKind, flashMsg
		factory := func() core.Component {
			page := &Page{
				Deps:          ui.deps,
				Views:         ui.views,
				Controller:    controllerFor(spec.screen),
				Screen:        spec.screen,
				Title:         title,
				Section:       spec.section,
				Protected:     spec.protected,
				RequiredPerms: append([]rbac.Permission(nil), spec.permission...),
				AccessToken:   access,
				RefreshToken:  refresh,
				Actor:         claims,
				ClientIP:      clientIP,
				UserAgent:     userAgent,
				CorrelationID: correlationID,
				Params:        cloneMap(params),
				Query:         cloneMap(query),
			}
			page.Locale = locale
			flashMu.Lock()
			k, m := pendingKind, pendingMsg
			pendingKind, pendingMsg = "", ""
			flashMu.Unlock()
			applyFlash(page, k, m)
			return page
		}
		if err := ui.registry.RegisterPage(name, factory, spec.mode); err != nil &&
			!errors.Is(err, core.ErrComponentAlreadyRegistered) {
			return err
		}
		ui.bindings.Store(name, componentBinding{
			protected:  spec.protected,
			tokenID:    claims.TokenID,
			permission: append([]rbac.Permission(nil), spec.permission...),
		})
		turnstileEnabled := ui.deps.Turnstile != nil && ui.deps.Turnstile.Enabled()
		opts := shellOpts{
			Title:            title,
			Component:        name,
			Locale:           locale,
			TurnstileEnabled: turnstileEnabled,
			Mode:             spec.mode,
			Head:             core.Head{Title: title, Lang: locale},
		}
		if spec.mode == core.ModeSEO || spec.mode == core.ModeStatic {
			body, head, err := ui.renderSSR(c.Context(), name, locale)
			if err != nil {
				return err
			}
			opts.Body = body
			opts.Head = head
		}
		return sendShell(c, opts)
	}
}

func (ui *UI) routeTitle(locale string, spec routeSpec) string {
	if spec.titleKey == "" || ui.deps.Translator == nil {
		return spec.title
	}
	return ui.deps.Translator.T(pkgi18n.Locale(locale), spec.titleKey, spec.title)
}

func (ui *UI) authenticate(c fiber.Ctx, access, refresh string) (appauth.Claims, string, string, error) {
	if ui.deps.Auth == nil {
		return appauth.Claims{}, "", "", errUnauthenticated
	}
	if access != "" {
		if claims, err := ui.deps.Auth.Verify(c.Context(), access); err == nil {
			setActorLocals(c, claims)
			return claims, access, refresh, nil
		}
	}
	if refresh == "" {
		return appauth.Claims{}, "", "", errUnauthenticated
	}
	tokens, err := ui.deps.Auth.Refresh(c.Context(), refresh)
	if err != nil {
		clearAuthCookies(c, ui.deps.Secure)
		return appauth.Claims{}, "", "", errUnauthenticated
	}
	setAuthCookies(c, tokens, ui.deps.Secure, ui.deps.AccessTTL)
	claims, err := ui.deps.Auth.Verify(c.Context(), tokens.AccessToken)
	if err != nil {
		return appauth.Claims{}, "", "", errUnauthenticated
	}
	setActorLocals(c, claims)
	return claims, tokens.AccessToken, tokens.RefreshToken, nil
}

func (ui *UI) requireUploadAccess(c fiber.Ctx) error {
	claims, _, _, err := ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
	if err != nil {
		return fiber.ErrUnauthorized
	}
	if !ui.anyPermission(c, claims.Role, []rbac.Permission{rbac.PermUploadsCreate, rbac.PermNotificationsSend}) {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func requireSameOrigin(c fiber.Ctx) error {
	origin := strings.TrimSpace(c.Get("Origin"))
	if origin == "" {
		return c.Next()
	}
	parsed, err := url.Parse(origin)
	if err != nil || !strings.EqualFold(parsed.Hostname(), c.Hostname()) {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func (ui *UI) authorizeWebSocket(c fiber.Ctx) error {
	component := c.Query("component")
	if component == "" {
		// Reconnects carry only the cryptographically random GoUI session id.
		return c.Next()
	}
	value, ok := ui.bindings.Load(component)
	if !ok {
		return fiber.ErrForbidden
	}
	binding := value.(componentBinding)
	if !binding.protected {
		return c.Next()
	}
	claims, _, _, err := ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
	if err != nil || claims.TokenID == "" ||
		subtle.ConstantTimeCompare([]byte(claims.TokenID), []byte(binding.tokenID)) != 1 {
		return fiber.ErrUnauthorized
	}
	if len(binding.permission) > 0 && !ui.anyPermission(c, claims.Role, binding.permission) {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func (ui *UI) anyPermission(c fiber.Ctx, role string, permissions []rbac.Permission) bool {
	for _, permission := range permissions {
		ok, err := ui.deps.Checker.Allows(c.Context(), role, permission)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func pageRoutes() []routeSpec {
	return []routeSpec{
		{path: "/", screen: "home", title: "GoCore", mode: core.ModeSEO},
		{path: "/contact", screen: "contact", title: "İletişim", mode: core.ModeSEO},
		{path: "/iletisim", screen: "contact", title: "İletişim", mode: core.ModeSEO},
		{path: "/auth/login", screen: "login", title: "Giriş"},
		{path: "/auth/register", screen: "register", title: "Kayıt"},
		{path: "/auth/forgot-password", screen: "forgot", title: "Şifremi Unuttum"},
		{path: "/auth/reset-password", screen: "reset", title: "Şifre Sıfırla"},
		{path: "/auth/verify-email", screen: "verify", title: "E-posta Doğrulama"},
		{path: "/dashboard", screen: "dashboard", title: "Dashboard", titleKey: "dashboard.title", section: "dashboard", protected: true},
		{path: "/dashboard/account", screen: "account", title: "Hesabım", titleKey: "dashboard.account.title", section: "account", protected: true},
		{path: "/dashboard/account/mfa", screen: "mfa", title: "İki Adımlı Doğrulama", titleKey: "dashboard.account.mfa", section: "account", protected: true},
		{path: "/dashboard/account/notifications", screen: "inbox", title: "Bildirimlerim", titleKey: "dashboard.account.notifications", section: "account", protected: true},
		{path: "/dashboard/uploads", screen: "uploads", title: "Dosya Yükleme", titleKey: "dashboard.uploads.title", section: "uploads", protected: true, permission: []rbac.Permission{rbac.PermUploadsCreate}},
		{path: "/dashboard/users", screen: "users", title: "Kullanıcılar", titleKey: "dashboard.users.title", section: "users", protected: true, permission: []rbac.Permission{rbac.PermUsersList}},
		{path: "/dashboard/users/new", screen: "user-new", title: "Yeni Kullanıcı", titleKey: "dashboard.users.new", section: "users", protected: true, permission: []rbac.Permission{rbac.PermUsersList}},
		{path: "/dashboard/users/:id", screen: "user-show", title: "Kullanıcı", titleKey: "dashboard.users.user", section: "users", protected: true},
		{path: "/dashboard/contacts", screen: "contacts", title: "İletişim", titleKey: "dashboard.contacts.title", section: "contacts", protected: true, permission: []rbac.Permission{rbac.PermContactsList}},
		{path: "/dashboard/contacts/:id", screen: "contact-show", title: "İletişim Mesajı", titleKey: "dashboard.contacts.detail", section: "contacts", protected: true, permission: []rbac.Permission{rbac.PermContactsList}},
		{path: "/dashboard/rbac/roles", screen: "roles", title: "Roller", titleKey: "dashboard.rbac.roles", section: "rbac", protected: true, permission: []rbac.Permission{rbac.PermRBACManage}},
		{path: "/dashboard/rbac/roles/new", screen: "role-new", title: "Yeni Rol", titleKey: "dashboard.rbac.new_role", section: "rbac", protected: true, permission: []rbac.Permission{rbac.PermRBACManage}},
		{path: "/dashboard/rbac/roles/:name", screen: "role-show", title: "Rol", titleKey: "dashboard.rbac.role", section: "rbac", protected: true, permission: []rbac.Permission{rbac.PermRBACManage}},
		{path: "/dashboard/rbac/permissions", screen: "permissions", title: "İzinler", titleKey: "dashboard.rbac.permissions", section: "rbac", protected: true, permission: []rbac.Permission{rbac.PermRBACManage}},
		{path: "/dashboard/notifications/send", screen: "notification-send", title: "Bildirim Gönder", titleKey: "dashboard.notifications.send", section: "notifications", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSend}},
		{path: "/dashboard/notifications/bulk", screen: "notification-bulk", title: "Toplu Bildirim", titleKey: "dashboard.notifications.bulk", section: "notifications", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSend}},
		{path: "/dashboard/notifications/bulk/upload", screen: "notification-upload", title: "Dosyadan Bildirim", titleKey: "dashboard.notifications.upload", section: "notifications", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSend}},
		{path: "/dashboard/settings/sms", screen: "sms-settings", title: "SMS Ayarları", titleKey: "dashboard.settings.sms.title", section: "settings-sms", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSettings}},
		{path: "/dashboard/settings/sms/:provider", screen: "sms-provider", title: "SMS Sağlayıcı", titleKey: "dashboard.settings.sms_provider", section: "settings-sms", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSettings}},
		{path: "/dashboard/settings/payment", screen: "payment-settings", title: "Ödeme Ayarları", titleKey: "dashboard.settings.payments.title", section: "settings-payment", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSettings}},
		{path: "/dashboard/settings/payment/:provider", screen: "payment-provider", title: "Ödeme Sağlayıcı", titleKey: "dashboard.settings.payment_provider", section: "settings-payment", protected: true, permission: []rbac.Permission{rbac.PermNotificationsSettings}},
		{path: "/dashboard/payments/checkout", screen: "checkout", title: "Ödeme", titleKey: "dashboard.payments.checkout", section: "payments-checkout", protected: true, permission: []rbac.Permission{rbac.PermPaymentsCharge}},
		{path: "/dashboard/payments/transactions", screen: "payments", title: "Ödemeler", titleKey: "dashboard.payments.title", section: "payments-list", protected: true, permission: []rbac.Permission{rbac.PermPaymentsList}},
		{path: "/dashboard/payments/transactions/:reference", screen: "payment-show", title: "Ödeme Detayı", titleKey: "dashboard.payments.detail", section: "payments-list", protected: true, permission: []rbac.Permission{rbac.PermPaymentsCharge, rbac.PermPaymentsList}},
		{path: "/dashboard/audit/logs", screen: "audit", title: "Audit Kayıtları", titleKey: "dashboard.audit.title", section: "audit", protected: true, permission: []rbac.Permission{rbac.PermAuditList}},
		{path: "/dashboard/audit/logs/:id", screen: "audit-show", title: "Audit Detayı", titleKey: "dashboard.audit.detail", section: "audit", protected: true, permission: []rbac.Permission{rbac.PermAuditList}},
	}
}

type shellOpts struct {
	Title            string
	Component        string
	Locale           string
	TurnstileEnabled bool
	Mode             core.PageMode
	Body             string
	Head             core.Head
}

func sendShell(c fiber.Ctx, opts shellOpts) error {
	locale := opts.Locale
	if locale == "" {
		locale = opts.Head.Lang
	}
	title := opts.Head.Title
	if title == "" {
		title = opts.Title
	}
	componentJSON, _ := json.Marshal(opts.Component)
	localeJSON, _ := json.Marshal(locale)
	turnstileHead := ""
	turnstileRuntime := ""
	observerTurnstile := ""
	if opts.TurnstileEnabled {
		turnstileHead = `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit" async defer></script>`
		turnstileRuntime = `const mountTurnstile=()=>{if(!window.turnstile)return;document.querySelectorAll(".cf-turnstile").forEach(el=>{const stamp=el.getAttribute("data-goui-ts")||"";const intact=!!el.querySelector("iframe");if(el.dataset.gouiMountedTs===stamp&&intact&&el.dataset.gouiRendered==="1")return;if(el.dataset.gouiWidgetId!=null){try{window.turnstile.remove(Number(el.dataset.gouiWidgetId));}catch{}delete el.dataset.gouiWidgetId;}el.replaceChildren();try{const id=window.turnstile.render(el,{sitekey:el.dataset.sitekey});el.dataset.gouiRendered="1";el.dataset.gouiMountedTs=stamp;if(id!=null)el.dataset.gouiWidgetId=String(id);}catch{}});};setInterval(mountTurnstile,500);`
		observerTurnstile = `mountTurnstile();`
	}
	alertRuntime := `const showAlert=(payload)=>{if(!window.Swal)return;const kind=(payload&&payload.kind)||"info";const text=(payload&&payload.text)||"";if(!text)return;const icons={success:"success",error:"error",warning:"warning",info:"info"};const titles={success:"Başarılı",error:"Hata",warning:"Uyarı",info:"Bilgi"};const icon=icons[kind]||"info";Swal.fire({icon,title:titles[kind]||"Bilgi",text,confirmButtonText:"Tamam",confirmButtonColor:kind==="error"?"#FF5A6B":"#11A8F4"});};` +
		`const FLASH_KEY="gocore.flash";const consumeFlashMarkers=(stashOnly)=>{document.querySelectorAll("[data-app-flash]").forEach(el=>{const kind=el.getAttribute("data-app-flash")||"info";const text=el.getAttribute("data-app-flash-msg")||"";if(!text)return;const sig=kind+"|"+text;if(el.dataset.gouiFlashSig===sig)return;el.dataset.gouiFlashSig=sig;if(stashOnly){sessionStorage.setItem(FLASH_KEY,JSON.stringify({kind,text}));return;}showAlert({kind,text});});};const restoreStashedFlash=()=>{const raw=sessionStorage.getItem(FLASH_KEY);if(!raw)return;sessionStorage.removeItem(FLASH_KEY);try{showAlert(JSON.parse(raw));}catch{}};const followRedirect=()=>{const meta=document.querySelector('#app meta[http-equiv="refresh"]');if(!meta)return;const target=(meta.content||"").replace(/^\d+\s*;\s*url=/i,"");if(!target)return;consumeFlashMarkers(true);window.location.assign(target);};restoreStashedFlash();consumeFlashMarkers(false);new MutationObserver(()=>{const redirecting=!!document.querySelector('#app meta[http-equiv="refresh"]');if(redirecting){followRedirect();` +
		observerTurnstile + `}else{consumeFlashMarkers(false);` + observerTurnstile + `}}).observe(document.getElementById("app"),{childList:true,subtree:true});`
	confirmRuntime := `document.addEventListener("submit",(ev)=>{const form=ev.target&&ev.target.closest?ev.target.closest("form[g-submit][data-confirm]"):null;if(!form)return;if(form.dataset.confirmed==="1"){delete form.dataset.confirmed;return;}ev.preventDefault();ev.stopImmediatePropagation();const msg=form.getAttribute("data-confirm")||"Bu işlem geri alınamaz.";const title=form.getAttribute("data-confirm-title")||"Emin misiniz?";const ok=form.getAttribute("data-confirm-ok")||"Evet, sil";const cancel=form.getAttribute("data-confirm-cancel")||"Vazgeç";const proceed=()=>{const client=window.__goui;const eventName=form.getAttribute("g-submit");const root=form.closest("[data-goui-component]");const componentId=root&&root.getAttribute("data-goui-component");if(client&&eventName&&componentId&&typeof client.sendEvent==="function"){const fields={};new FormData(form).forEach((value,key)=>{fields[key]=value;});client.sendEvent(componentId,eventName,{fields});return;}form.dataset.confirmed="1";if(typeof form.requestSubmit==="function")form.requestSubmit();else form.dispatchEvent(new Event("submit",{bubbles:true,cancelable:true}));};const run=()=>{if(!window.Swal){if(window.confirm(msg))proceed();return;}Swal.fire({icon:"warning",title,text:msg,showCancelButton:true,confirmButtonText:ok,cancelButtonText:cancel,confirmButtonColor:"#FF5A6B",cancelButtonColor:"#6D7C91",reverseButtons:true}).then((res)=>{if(res.isConfirmed)proceed();});};run();},true);`
	dashboardRuntime := `const closeSidebar=()=>document.body.classList.remove("sidebar-open");document.addEventListener("click",ev=>{if(ev.target.closest("[data-sidebar-toggle]"))document.body.classList.add("sidebar-open");if(ev.target.closest("[data-sidebar-close]"))closeSidebar();});document.addEventListener("keydown",ev=>{if(ev.key==="Escape")closeSidebar();});document.addEventListener("change",ev=>{const el=ev.target.closest("[data-lang-select]");if(!(el instanceof HTMLSelectElement)||!el.value)return;const next=encodeURIComponent(location.pathname+location.search+location.hash);window.location.assign("/lang/"+encodeURIComponent(el.value)+"?next="+next);},true);` +
		`(()=>{const STORE="gocore.notifUnread";let last=Number(sessionStorage.getItem(STORE));if(!Number.isFinite(last))last=-1;let writing=false;const bell=()=>document.querySelector("[data-notif-bell]");const badge=()=>document.querySelector("[data-notif-badge]");const apply=(n,animate)=>{const b=bell(),el=badge();if(!el||!b)return;const next=Math.max(0,n|0);const prev=Number(el.dataset.count);const unchanged=Number.isFinite(prev)&&prev===next;const grew=last>=0&&next>last;if(unchanged&&!animate){last=next;return;}const label=next>99?"99+":String(next);writing=true;el.dataset.count=String(next);if(next>0){el.hidden=false;el.textContent=label;b.setAttribute("aria-label",(b.getAttribute("title")||"Bildirimler")+" ("+label+")");}else{el.hidden=true;el.textContent="0";b.setAttribute("aria-label",b.getAttribute("title")||"Bildirimler");}writing=false;if(animate&&grew){b.classList.remove("notif-bell--ring");void b.offsetWidth;b.classList.add("notif-bell--ring");setTimeout(()=>b.classList.remove("notif-bell--ring"),900);}last=next;sessionStorage.setItem(STORE,String(next));};window.__gocoreNotif={apply};const syncFromDom=()=>{if(writing)return;const el=badge();if(!el)return;const n=Number(el.dataset.count);if(Number.isFinite(n))apply(n,false);};syncFromDom();const root=document.getElementById("app");if(root)new MutationObserver(()=>{if(!writing)syncFromDom();}).observe(root,{childList:true,subtree:true});})();`
	realtimeRuntime := `(()=>{let attempt=0,timer=null,ws=null;const url=()=>(location.protocol==="https:"?"wss://":"ws://")+location.host+"/api/v1/ws";const handle=(raw)=>{try{const msg=JSON.parse(raw);if(msg&&msg.type==="inbox.updated"&&window.__gocoreNotif)window.__gocoreNotif.apply(Number(msg.unread_count)||0,true);}catch{}};const connect=()=>{if(!document.querySelector("[data-notif-bell]"))return;try{ws=new WebSocket(url());}catch{schedule();return;}ws.onopen=()=>{attempt=0;};ws.onmessage=(ev)=>handle(ev.data);ws.onclose=()=>{ws=null;schedule();};ws.onerror=()=>{try{ws&&ws.close();}catch{}};};const schedule=()=>{if(timer)return;const delay=Math.min(30000,1000*Math.pow(1.5,attempt++));timer=setTimeout(()=>{timer=null;connect();},delay);};const boot=()=>{if(document.querySelector("[data-notif-bell]")){connect();return true;}return false;};if(!boot()){const root=document.getElementById("app");if(root){const mo=new MutationObserver(()=>{if(boot())mo.disconnect();});mo.observe(root,{childList:true,subtree:true});}}})();`
	onPushRuntime := `const onPush=(payload)=>showAlert(payload);`
	lang := opts.Head.Lang
	if lang == "" {
		lang = locale
	}
	doc := `<!doctype html><html lang="` + html.EscapeString(lang) + `"><head><meta charset="utf-8">` +
		`<meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>` + html.EscapeString(title) + `</title>` +
		renderHeadMeta(opts.Head) +
		`<link rel="preconnect" href="https://fonts.bunny.net"><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>` +
		`<link href="https://fonts.bunny.net/css?family=figtree:400,500,600,700&display=swap" rel="stylesheet">` +
		`<link href="https://fonts.googleapis.com/css2?family=Audiowide&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">` +
		`<link rel="stylesheet" href="/forms/style.css"><link rel="stylesheet" href="/goui/assets/app.css?v=account-settings-bg">` +
		`<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/sweetalert2@11/dist/sweetalert2.min.css">` +
		turnstileHead +
		`</head><body><div id="app" aria-live="polite">` + opts.Body + `</div>`
	if opts.Mode != core.ModeStatic {
		doc += `<script src="https://cdn.jsdelivr.net/npm/sweetalert2@11/dist/sweetalert2.all.min.js"></script>` +
			`<script type="module">import {GoUIClient} from "/client/goui.js";import {enhanceUpload} from "/client/modules/upload.js";` +
			`sessionStorage.removeItem("goui.sessionId");enhanceUpload(document);` + turnstileRuntime + alertRuntime + confirmRuntime + dashboardRuntime + realtimeRuntime + onPushRuntime +
			`window.__goui=new GoUIClient("/goui/ws",` + string(componentJSON) + `,{mount:"#app",locale:` + string(localeJSON) + `,onPush:onPush});window.__goui.connect();</script>`
	}
	doc += `</body></html>`
	c.Type("html", "utf-8")
	return c.SendString(doc)
}

func componentName(spec routeSpec, claims appauth.Claims, csrf string, params, query map[string]string) string {
	raw, _ := json.Marshal([]any{spec.screen, claims.TokenID, claims.UserID, csrf, params, query})
	sum := sha256.Sum256(raw)
	return "page:" + spec.screen + ":" + hex.EncodeToString(sum[:16])
}

func routeParams(c fiber.Ctx) map[string]string {
	out := map[string]string{}
	for _, name := range []string{"id", "name", "provider", "reference"} {
		if value := c.Params(name); value != "" {
			out[name] = value
		}
	}
	return out
}

func routeQuery(c fiber.Ctx) map[string]string {
	out := map[string]string{}
	for key, value := range c.Request().URI().QueryArgs().All() {
		out[string(key)] = string(value)
	}
	return out
}

func ensureCSRFCookie(c fiber.Ctx, secure bool) string {
	// csrf_token, klasik double-submit CSRF değil: component adı hash'ine
	// bağlanır (tokenID + route). Mutasyon koruması EventNonce + same-origin +
	// WS tokenID binding ile sağlanır.
	if token := c.Cookies(cookieCSRF); token != "" {
		return token
	}
	var data [32]byte
	_, _ = rand.Read(data[:])
	token := base64.RawURLEncoding.EncodeToString(data[:])
	c.Cookie(&fiber.Cookie{
		Name: cookieCSRF, Value: token, Path: "/", Secure: secure,
		HTTPOnly: false, SameSite: fiber.CookieSameSiteStrictMode,
	})
	return token
}

func setActorLocals(c fiber.Ctx, claims appauth.Claims) {
	c.Locals(adapters.LocalUserID, claims.UserID)
	c.Locals(adapters.LocalRole, claims.Role)
	c.Locals(adapters.LocalEmail, claims.Email)
	c.SetContext(logger.WithUserID(c.Context(), claims.UserID))
}

func setAuthCookies(c fiber.Ctx, tokens appauth.TokenPair, secure bool, accessTTL time.Duration) {
	maxAge := int(accessTTL.Seconds())
	if maxAge <= 0 {
		maxAge = int((15 * time.Minute).Seconds())
	}
	c.Cookie(&fiber.Cookie{Name: cookieAccess, Value: tokens.AccessToken, HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteStrictMode, Path: "/", MaxAge: maxAge})
	c.Cookie(&fiber.Cookie{Name: cookieRefresh, Value: tokens.RefreshToken, HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteStrictMode, Path: "/", MaxAge: int((7 * 24 * time.Hour).Seconds())})
}

func clearAuthCookies(c fiber.Ctx, secure bool) {
	for _, name := range []string{cookieAccess, cookieRefresh} {
		c.Cookie(&fiber.Cookie{Name: name, Value: "", Expires: time.Now().Add(-time.Hour), HTTPOnly: true, Secure: secure, SameSite: fiber.CookieSameSiteStrictMode, Path: "/"})
	}
}

func (ui *UI) localeFromRequest(c fiber.Ctx) string {
	explicit := strings.TrimSpace(c.Query("lang"))
	if explicit == "" {
		explicit = c.Cookies(adapters.LangCookie)
	}
	accept := c.Get(fiber.HeaderAcceptLanguage)
	if ui != nil && ui.deps.Translator != nil {
		return string(ui.deps.Translator.Resolve(explicit, accept))
	}
	if explicit != "" {
		return explicit
	}
	if locale, _ := c.Locals(adapters.LocalLocale).(string); locale != "" {
		return locale
	}
	return "tr"
}

func isGuestScreen(screen string) bool {
	switch screen {
	case "login", "register", "forgot", "reset", "verify":
		return true
	default:
		return false
	}
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func randomToken(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
