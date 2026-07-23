package goui

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/ws"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	pkgi18n "github.com/zatrano/gocore/pkg/i18n"
)

type probeController struct {
	events int
}

func (*probeController) Mount(context.Context, *Page) error { return nil }

func (*probeController) Render(*Page) (string, error) {
	return `<section><form g-submit="save"><input name="name"></form></section>`, nil
}

func (p *probeController) HandleEvent(_ context.Context, _ *Page, _ string, _ map[string]any) error {
	p.events++
	return nil
}

func TestPageLifecycleInjectsAndConsumesEventNonce(t *testing.T) {
	controller := &probeController{}
	page := &Page{Controller: controller}
	if err := page.Mount(context.Background()); err != nil {
		t.Fatal(err)
	}
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `name="_goui_nonce"`) || !strings.Contains(rendered, `g-submit="save"`) {
		t.Fatalf("submit security field missing: %s", rendered)
	}

	nonce := page.EventNonce
	payload := map[string]any{"fields": map[string]any{"_goui_nonce": nonce, "name": "Ada"}}
	if err := page.HandleEvent(context.Background(), "save", payload); err != nil {
		t.Fatal(err)
	}
	if controller.events != 1 {
		t.Fatalf("events=%d", controller.events)
	}
	if err := page.HandleEvent(context.Background(), "save", payload); err != nil {
		t.Fatal(err)
	}
	if controller.events != 1 || page.Error == "" {
		t.Fatalf("replayed event was not rejected: events=%d error=%q", controller.events, page.Error)
	}
}

func TestProtectedPageRendersDashboardShell(t *testing.T) {
	page := &Page{
		Controller: &probeController{},
		Protected:  true,
		Title:      "Dashboard",
		Section:    "dashboard",
		EventNonce: "nonce",
	}
	page.Locale = "tr"
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`class="gocore-shell panel-shell"`,
		`class="shell-panel-sidebar"`,
		`class="gocore-header shell-panel-topbar"`,
		`sidebar-nav-link--active`,
		`data-lang-select`,
		`value="tr" selected`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("dashboard shell is missing %q: %s", expected, rendered)
		}
	}
}

func TestPublicPageRendersLanguageSwitch(t *testing.T) {
	page := &Page{
		Controller: &probeController{},
		Screen:     "home",
		EventNonce: "nonce",
	}
	page.Locale = "en"
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`class="lang-switch"`,
		`data-lang-select`,
		`value="tr"`,
		`value="en" selected`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("public language switch is missing %q: %s", expected, rendered)
		}
	}
}

func TestPublicHeaderRegisterUsesLocale(t *testing.T) {
	tr, err := pkgi18n.NewFromEmbedded("tr", []pkgi18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	page := &Page{
		Controller: &probeController{},
		Screen:     "home",
		EventNonce: "nonce",
		Deps:       Deps{Translator: tr, Locales: []string{"tr", "en"}},
	}
	page.Locale = "en"
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `href="/auth/register">Sign up</a>`) {
		t.Fatalf("register button not translated: %s", rendered)
	}
	if strings.Contains(rendered, `href="/auth/register">Kayıt</a>`) {
		t.Fatal("register button still shows Turkish label in EN locale")
	}
}

func TestPublicHeaderShowsPanelWhenAuthenticated(t *testing.T) {
	tr, err := pkgi18n.NewFromEmbedded("tr", []pkgi18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	page := &Page{
		Controller: &probeController{},
		Screen:     "home",
		EventNonce: "nonce",
		Deps:       Deps{Translator: tr, Locales: []string{"tr", "en"}},
		Actor:      appauth.Claims{UserID: "u1"},
	}
	page.Locale = "en"
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `href="/dashboard">Dashboard</a>`) {
		t.Fatalf("expected panel button for authenticated user: %s", rendered)
	}
	if strings.Contains(rendered, `href="/auth/login"`) || strings.Contains(rendered, `href="/auth/register"`) {
		t.Fatalf("login/register must be hidden when authenticated: %s", rendered)
	}
}

func TestGuestPageRendersAuthShell(t *testing.T) {
	page := &Page{
		Controller: &probeController{},
		Screen:     "login",
		Title:      "Giriş",
		EventNonce: "nonce",
	}
	rendered, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`class="gocore-shell auth-shell"`,
		`class="auth-shell-backdrop"`,
		`class="auth-brand"`,
		`data-lang-select`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("auth shell is missing %q: %s", expected, rendered)
		}
	}
	if strings.Contains(rendered, `class="gocore-header"`) {
		t.Fatalf("auth shell must not render public header: %s", rendered)
	}
}

func TestPageHeadProvidesSEOMeta(t *testing.T) {
	page := &Page{
		Controller: &probeController{},
		Screen:     "home",
		Title:      "GoCore",
		EventNonce: "nonce",
	}
	page.Locale = "tr"
	head := page.Head()
	if head.Title != "GoCore" || head.Description == "" || head.OGTitle == "" {
		t.Fatalf("unexpected head: %+v", head)
	}
}

func TestHomeRouteServesModeSEOBody(t *testing.T) {
	ui, err := New(Deps{})
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	app := fiber.New()
	ui.Register(app)

	response, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	htmlOut := string(body)
	for _, expected := range []string{
		`data-goui-ssr="1"`,
		`name="description"`,
		`property="og:title"`,
		`id="app"`,
		`GoUIClient`,
		`Kurumsal İş Yönetim Platformu`,
	} {
		if !strings.Contains(htmlOut, expected) {
			t.Fatalf("ModeSEO home missing %q (len=%d)", expected, len(htmlOut))
		}
	}
}

func TestLoginRouteStaysModeLiveEmptyApp(t *testing.T) {
	ui, err := New(Deps{})
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	app := fiber.New()
	ui.Register(app)

	response, err := app.Test(httptest.NewRequest("GET", "/auth/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	htmlOut := string(body)
	if strings.Contains(htmlOut, `data-goui-ssr`) {
		t.Fatal("ModeLive login must not embed SSR markers")
	}
	if !strings.Contains(htmlOut, `<div id="app" aria-live="polite"></div>`) {
		t.Fatalf("ModeLive login should serve empty #app: %s", htmlOut[:min(400, len(htmlOut))])
	}
}

func TestAssetRoutesServeGoUIRuntime(t *testing.T) {
	app := fiber.New()
	registerAssetRoutes(app)
	response, err := app.Test(httptest.NewRequest("GET", "/client/goui.js", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !bytes.Contains(body, []byte("class GoUIClient")) {
		t.Fatal("GoUI runtime was not served")
	}
}

func TestRequireSameOriginRejectsCrossSiteWebSocket(t *testing.T) {
	app := fiber.New()
	app.Use(requireSameOrigin)
	app.Get("/goui/ws", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	request := httptest.NewRequest("GET", "http://app.example/goui/ws", nil)
	request.Header.Set("Origin", "https://evil.example")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestWebSocketComponentMustBeIssuedByPageRoute(t *testing.T) {
	ui, err := New(Deps{})
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	app := fiber.New()
	app.Use(ui.authorizeWebSocket)
	app.Get("/goui/ws", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	unknown, err := app.Test(httptest.NewRequest("GET", "/goui/ws?component=unknown", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer unknown.Body.Close()
	if unknown.StatusCode != fiber.StatusForbidden {
		t.Fatalf("unknown status=%d", unknown.StatusCode)
	}

	ui.bindings.Store("public-page", componentBinding{})
	known, err := app.Test(httptest.NewRequest("GET", "/goui/ws?component=public-page", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer known.Body.Close()
	if known.StatusCode != fiber.StatusNoContent {
		t.Fatalf("known status=%d", known.StatusCode)
	}
}

func TestEveryPageRouteHasAConcreteController(t *testing.T) {
	for _, route := range pageRoutes() {
		controller := controllerFor(route.screen)
		if _, missing := controller.(*missingController); missing {
			t.Errorf("%s (%s) has no GoUI controller", route.path, route.screen)
		}
	}
}

type memoryWSConn struct {
	mu       sync.Mutex
	writes   [][]byte
	rendered chan struct{}
	once     sync.Once
}

func (c *memoryWSConn) ReadMessage() (int, []byte, error) {
	select {
	case <-c.rendered:
	case <-time.After(2 * time.Second):
	}
	return 0, nil, io.EOF
}
func (c *memoryWSConn) WriteMessage(_ int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, bytes.Clone(data))
	if bytes.Contains(data, []byte(`"type":"render"`)) {
		c.once.Do(func() { close(c.rendered) })
	}
	return nil
}
func (*memoryWSConn) Close() error { return nil }

func TestGoUIWebSocketServerMountsAndRendersComponent(t *testing.T) {
	ui, err := New(Deps{})
	if err != nil {
		t.Fatal(err)
	}
	defer ui.Close()
	if err := ui.registry.Register("probe", func() core.Component {
		return &Page{Controller: &probeController{}, Views: ui.views}
	}); err != nil {
		t.Fatal(err)
	}
	conn := &memoryWSConn{rendered: make(chan struct{})}
	if err := ui.server.ServeConn(context.Background(), conn, ws.ConnectParams{
		ComponentName: "probe",
		Locale:        "tr",
	}); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	joined := bytes.Join(conn.writes, nil)
	if !bytes.Contains(joined, []byte(`"type":"session"`)) ||
		!bytes.Contains(joined, []byte(`"type":"render"`)) {
		t.Fatalf("expected session and render frames, got %s", joined)
	}
}

type memoryStorage struct {
	files map[string][]byte
	meta  map[string]appshared.FileObject
}

func (m *memoryStorage) Put(_ context.Context, key string, r io.Reader, contentType string, size int64) (appshared.FileObject, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return appshared.FileObject{}, err
	}
	obj := appshared.FileObject{Key: key, ContentType: contentType, Size: int64(len(data))}
	m.files[key], m.meta[key] = data, obj
	return obj, nil
}

func (m *memoryStorage) Get(_ context.Context, key string) (io.ReadCloser, appshared.FileObject, error) {
	return io.NopCloser(bytes.NewReader(m.files[key])), m.meta[key], nil
}

func (m *memoryStorage) Delete(_ context.Context, key string) error {
	delete(m.files, key)
	delete(m.meta, key)
	return nil
}

func TestUploadStoreBridgesApplicationStorage(t *testing.T) {
	storage := &memoryStorage{files: map[string][]byte{}, meta: map[string]appshared.FileObject{}}
	store := NewUploadStore(storage, 1024, []string{"text/plain"})
	meta, err := store.Save("../note.txt", "text/plain", strings.NewReader("hello"), 5)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "note.txt" || meta.ID == "" || meta.URL == "" {
		t.Fatalf("meta=%+v", meta)
	}
	reader, opened, err := store.Open(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if opened.ContentType != "text/plain" {
		t.Fatalf("content type=%q", opened.ContentType)
	}
}
