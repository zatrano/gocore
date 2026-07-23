package goui

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/cache"
	pkgi18n "github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/validation"
)

func testPublicAuthValidate() *validator.Validate {
	v := validator.New()
	validation.Register(v)
	return v
}

func TestPublicAuthControllerFactory(t *testing.T) {
	t.Parallel()
	for _, screen := range []string{"home", "contact", "login", "register", "forgot", "reset", "verify"} {
		if publicAuthController(screen) == nil {
			t.Fatalf("publicAuthController(%q) = nil", screen)
		}
	}
	if publicAuthController("dashboard") != nil {
		t.Fatal("expected nil for unknown screen")
	}
}

func TestHomeAndLoginRender(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	home := publicAuthController("home")
	page := &Page{Controller: home, Deps: Deps{Validate: testPublicAuthValidate()}}
	if err := home.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := home.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Kurumsal İş Yönetim Platformu", `data-key="users"`, `href="/auth/login"`, `href="/iletisim"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("home render missing %q", want)
		}
	}

	login := publicAuthController("login").(*loginController)
	login.email = `<script>alert(1)</script>`
	page = &Page{Controller: login, Deps: Deps{Validate: testPublicAuthValidate()}, Query: map[string]string{"next": "/dashboard/users"}}
	if err := login.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err = login.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `g-submit="login.submit"`) {
		t.Fatal("login form must use g-submit")
	}
	for _, want := range []string{
		`class="form-row"`,
		`id="login-email"`,
		`id="login-password"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("login form missing shared field markup %q: %s", want, html)
		}
	}
	if strings.Contains(html, `<script>alert(1)</script>`) {
		t.Fatal("email must be HTML-escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("expected escaped script markers in email value")
	}
}

func TestLoginAndRegisterRenderOAuthButtons(t *testing.T) {
	t.Parallel()
	tr, err := pkgi18n.NewFromEmbedded("tr", []pkgi18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}

	login := &loginController{providers: []string{"google", "github"}, next: "/dashboard"}
	page := &Page{Controller: login, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
	page.Locale = "en"
	html, err := login.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="auth-oauth-section" aria-label="OAuth"`,
		`class="oauth-list"`,
		`class="oauth-btn oauth-btn--google"`,
		`href="/auth/oauth/google?from=login&amp;next=%2Fdashboard"`,
		`Continue with Google`,
		`Continue with GitHub`,
		`<strong>or</strong>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("login oauth missing %q: %s", want, html)
		}
	}

	register := &registerController{providers: []string{"google"}}
	page = &Page{Controller: register, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
	page.Locale = "tr"
	html, err = register.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="oauth-btn oauth-btn--google"`,
		`href="/auth/oauth/google?from=register"`,
		`Google ile Kayıt Ol`,
		`<strong>veya</strong>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("register oauth missing %q: %s", want, html)
		}
	}
}

func TestAuthFooterLinkLayout(t *testing.T) {
	t.Parallel()

	login := &loginController{}
	html, err := login.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<nav class="auth-links"`,
		`href="/auth/forgot-password">Şifremi unuttum</a>`,
		`href="/auth/register">Kayıt</a>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("login footer missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, " · ") {
		t.Fatalf("login links must not use an inline separator: %s", html)
	}

	for name, ctrl := range map[string]Controller{
		"register": &registerController{},
		"forgot":   &forgotController{},
		"reset":    &resetController{},
		"verify":   &verifyController{},
	} {
		html, err = ctrl.Render(&Page{})
		if err != nil {
			t.Fatalf("%s render: %v", name, err)
		}
		if !strings.Contains(html, `class="auth-links auth-links-single"`) {
			t.Fatalf("%s single link must be centered: %s", name, html)
		}
	}
}

func TestContactValidationEvent(t *testing.T) {
	t.Parallel()
	ctrl := publicAuthController("contact").(*contactController)
	page := &Page{Controller: ctrl, Deps: Deps{Validate: testPublicAuthValidate()}}
	err := ctrl.HandleEvent(context.Background(), page, "contact.submit", map[string]any{
		"fields": map[string]any{
			"name":    "A",
			"email":   "not-an-email",
			"message": "hi",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if ctrl.fieldErrors == nil {
		t.Fatal("expected field errors")
	}
	html, renderErr := ctrl.Render(page)
	if renderErr != nil {
		t.Fatal(renderErr)
	}
	if !strings.Contains(html, `g-submit="contact.submit"`) {
		t.Fatal("contact form must use g-submit")
	}
	for _, want := range []string{
		`id="contact-name"`,
		`id="contact-email"`,
		`id="contact-message" name="message"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("contact form missing shared field markup %q: %s", want, html)
		}
	}
}

func TestLoginValidationAndCacheRequired(t *testing.T) {
	t.Parallel()
	ctrl := publicAuthController("login").(*loginController)
	page := &Page{Controller: ctrl, Deps: Deps{Validate: testPublicAuthValidate()}}

	err := ctrl.HandleEvent(context.Background(), page, "login.submit", map[string]any{
		"fields": map[string]any{"email": "", "password": ""},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	page.Deps.Cache = nil
	err = redirectWithAuthExchange(context.Background(), page, appauth.TokenPair{
		AccessToken: "a", RefreshToken: "r",
	}, "/dashboard")
	if !errors.Is(err, errAuthCacheUnavailable) {
		t.Fatalf("got %v, want errAuthCacheUnavailable", err)
	}
}

func TestLoginMFARender(t *testing.T) {
	t.Parallel()
	ctrl := &loginController{
		email: "user@example.com", mfaChallenge: "opaque-challenge", mfaRequired: true, next: "/dashboard",
	}
	html, err := ctrl.Render(&Page{Deps: Deps{Validate: testPublicAuthValidate()}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "İki Adımlı Doğrulama") {
		t.Fatal("expected MFA title")
	}
	if !strings.Contains(html, `name="mfa_code"`) {
		t.Fatal("expected mfa_code field")
	}
	if !strings.Contains(html, `name="mfa_challenge" value="opaque-challenge"`) {
		t.Fatal("expected opaque MFA challenge")
	}
	if strings.Contains(html, `name="password"`) || strings.Contains(html, `name="email"`) {
		t.Fatal("MFA step must not contain password or email")
	}
}

func TestAuthExchangeStoreConsume(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mem := cache.NewMemory()
	tokens := appauth.TokenPair{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer"}

	code, err := storeAuthExchange(ctx, mem, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if code == "" {
		t.Fatal("empty code")
	}

	got, err := consumeAuthExchange(ctx, mem, code)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" {
		t.Fatalf("unexpected tokens: %+v", got)
	}

	if _, err := consumeAuthExchange(ctx, mem, code); !errors.Is(err, errAuthExchangeInvalid) {
		t.Fatalf("second consume: got %v, want invalid", err)
	}
}

func TestRedirectWithAuthExchangeSetsPage(t *testing.T) {
	t.Parallel()
	page := &Page{Deps: Deps{Cache: cache.NewMemory()}}
	err := redirectWithAuthExchange(context.Background(), page, appauth.TokenPair{
		AccessToken: "a", RefreshToken: "r",
	}, "/dashboard/account")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(page.Redirect, "/goui/auth/exchange?code=") {
		t.Fatalf("redirect = %q", page.Redirect)
	}
	if !strings.Contains(page.Redirect, "next=%2Fdashboard%2Faccount") {
		t.Fatalf("missing next in %q", page.Redirect)
	}
}

func TestUtilityAuthExchangeRoute(t *testing.T) {
	t.Parallel()
	mem := cache.NewMemory()
	code, err := storeAuthExchange(context.Background(), mem, appauth.TokenPair{
		AccessToken: "access-token", RefreshToken: "refresh-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	ui := &UI{deps: Deps{Cache: mem, Secure: false, AccessTTL: time.Minute}}
	app := fiber.New()
	ui.registerUtilityRoutes(app)

	req := httptest.NewRequest(fiber.MethodGet, "/goui/auth/exchange?code="+code+"&next=/dashboard", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound && resp.StatusCode != fiber.StatusSeeOther && resp.StatusCode != 302 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/dashboard" {
		t.Fatalf("Location = %q", loc)
	}
	cookies := resp.Header.Values("Set-Cookie")
	joined := strings.Join(cookies, "\n")
	if !strings.Contains(joined, "access_token=") || !strings.Contains(joined, "refresh_token=") {
		t.Fatalf("missing auth cookies: %v", cookies)
	}
	if !strings.Contains(joined, "web_flash_type=") || !strings.Contains(joined, "web_flash_msg=") {
		t.Fatalf("missing flash cookies: %v", cookies)
	}
}

func TestUtilityPaymentRoutesNotRegisteredOnGoUI(t *testing.T) {
	t.Parallel()
	ui := &UI{deps: Deps{}}
	app := fiber.New()
	ui.registerUtilityRoutes(app)

	for _, path := range []string{"/payments/3ds/callback", "/payments/webhook/iyzico"} {
		req := httptest.NewRequest(fiber.MethodPost, path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != fiber.StatusNotFound && resp.StatusCode != fiber.StatusMethodNotAllowed {
			t.Fatalf("POST %s status = %d, want 404 (canonical /api/v1/...)", path, resp.StatusCode)
		}
	}
}

func TestLocalizedContactPaths(t *testing.T) {
	t.Parallel()
	if got := contactPath("tr"); got != "/iletisim" {
		t.Fatalf("contactPath(tr)=%q", got)
	}
	if got := contactPath("en"); got != "/contact" {
		t.Fatalf("contactPath(en)=%q", got)
	}
	if got := localizedPath("/iletisim", "en"); got != "/contact" {
		t.Fatalf("localizedPath iletisim→en = %q", got)
	}
	if got := localizedPath("/contact", "tr"); got != "/iletisim" {
		t.Fatalf("localizedPath contact→tr = %q", got)
	}
}

func TestLanguageSwitchRewritesContactPath(t *testing.T) {
	t.Parallel()
	ui := &UI{deps: Deps{Locales: []string{"tr", "en"}}}
	app := fiber.New()
	ui.registerUtilityRoutes(app)

	req := httptest.NewRequest(fiber.MethodGet, "/lang/en?next=/iletisim", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusFound && resp.StatusCode != fiber.StatusSeeOther && resp.StatusCode != 302 && resp.StatusCode != 303 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/contact" {
		t.Fatalf("Location = %q, want /contact", loc)
	}
}

func TestHomeAndContactRenderUseLocale(t *testing.T) {
	t.Parallel()
	tr, err := pkgi18n.NewFromEmbedded("tr", []pkgi18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	home := publicAuthController("home")
	page := &Page{Controller: home, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
	page.Locale = "en"
	html, err := home.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Enterprise Business Management Platform", `href="/contact"`, "Contact"} {
		if !strings.Contains(html, want) {
			t.Fatalf("en home missing %q in %s", want, html)
		}
	}

	contact := publicAuthController("contact")
	page = &Page{Controller: contact, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
	page.Locale = "en"
	html, err = contact.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1>Contact</h1>", "Full Name", "Send"} {
		if !strings.Contains(html, want) {
			t.Fatalf("en contact missing %q in %s", want, html)
		}
	}
}

func TestAuthPagesRenderUseLocale(t *testing.T) {
	t.Parallel()
	tr, err := pkgi18n.NewFromEmbedded("tr", []pkgi18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		screen string
		want   []string
	}{
		{"login", []string{"<h1>Sign in</h1>", "Email", "Password", "Forgot password", "Sign up"}},
		{"register", []string{"<h1>Sign up</h1>", "Full name", "Phone", "Sign in"}},
		{"forgot", []string{"<h1>Forgot password</h1>", "Send reset link", "Back to sign in"}},
		{"reset", []string{"<h1>Reset password</h1>", "New password", "Back to sign in"}},
		{"verify", []string{"<h1>Email verification</h1>", "Send verification email", "Back to sign in"}},
	}
	for _, tc := range cases {
		ctrl := publicAuthController(tc.screen)
		page := &Page{Controller: ctrl, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
		page.Locale = "en"
		html, err := ctrl.Render(page)
		if err != nil {
			t.Fatalf("%s: %v", tc.screen, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(html, want) {
				t.Fatalf("en %s missing %q in %s", tc.screen, want, html)
			}
		}
	}

	login := publicAuthController("login").(*loginController)
	login.mfaRequired = true
	page := &Page{Controller: login, Deps: Deps{Translator: tr, Locales: []string{"tr", "en"}}}
	page.Locale = "en"
	html, err := login.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1>Two-factor authentication</h1>", "Authenticator or recovery code", "Verify"} {
		if !strings.Contains(html, want) {
			t.Fatalf("en mfa login missing %q in %s", want, html)
		}
	}
}

func TestLoginRedirectURL(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":             "/dashboard",
		"/dashboard":   "/dashboard",
		"/auth/login":  "/dashboard",
		"https://evil": "/dashboard",
		"//evil":       "/dashboard",
		"/users":       "/users",
	}
	for in, want := range cases {
		if got := loginRedirectURL(in); got != want {
			t.Fatalf("loginRedirectURL(%q)=%q want %q", in, got, want)
		}
	}
}

func TestPayloadFieldsContract(t *testing.T) {
	t.Parallel()
	ctrl := publicAuthController("forgot").(*forgotController)
	page := &Page{Controller: ctrl, Deps: Deps{Validate: testPublicAuthValidate()}}
	err := ctrl.HandleEvent(context.Background(), page, "forgot.submit", map[string]any{
		"fields": map[string]any{"email": "user@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Notice == "" {
		t.Fatal("expected success notice")
	}
	if ctrl.email != "user@example.com" {
		t.Fatalf("email = %q", ctrl.email)
	}
}
