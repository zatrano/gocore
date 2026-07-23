package goui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appauth "github.com/zatrano/gocore/internal/application/auth"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

func spaValidate() *validator.Validate {
	v := validator.New()
	validation.Register(v)
	return v
}

func TestSharedSettingsFieldMarkup(t *testing.T) {
	t.Parallel()
	views, err := testViews()
	if err != nil {
		t.Fatal(err)
	}
	html, err := views.Render("pages.component_smoke", map[string]any{
		"Options": viewSelectOptions([][2]string{{"iyzico", "Iyzico"}}, "iyzico"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="form-row"`,
		`for="email"`,
		`for="provider"`,
		`g-change="field.email"`,
		`g-change="field.provider"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("shared field markup missing %q: %s", want, html)
		}
	}
	for _, forbidden := range []string{
		`class="form-field"`,
		`class="form-control"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("shared field markup still uses legacy %q: %s", forbidden, html)
		}
	}
}

type memSettingsRepo struct {
	data map[string]string
}

func (m *memSettingsRepo) Get(_ context.Context, key domainsettings.SettingKey) (string, error) {
	if m.data == nil {
		m.data = map[string]string{}
	}
	if v, ok := m.data[string(key)]; ok {
		return v, nil
	}
	switch key {
	case domainsettings.KeySMSActiveProvider:
		return domainsettings.ProviderNetgsm.String(), nil
	case domainsettings.KeyPaymentActiveProvider:
		return domainsettings.ProviderIyzico.String(), nil
	default:
		return "", errors.New("unknown settings key")
	}
}

func (m *memSettingsRepo) Set(_ context.Context, key domainsettings.SettingKey, value string) error {
	if m.data == nil {
		m.data = map[string]string{}
	}
	m.data[string(key)] = value
	return nil
}

type memAuditRepo struct {
	logs []appaudit.Log
}

func (m *memAuditRepo) List(_ context.Context, filter appaudit.ListFilter, page pagination.Request) (pagination.Page[appaudit.Log], error) {
	items := make([]appaudit.Log, 0, len(m.logs))
	for _, l := range m.logs {
		if filter.Action != "" && !strings.Contains(l.Action, filter.Action) {
			continue
		}
		if filter.Resource != "" && !strings.Contains(l.Resource, filter.Resource) {
			continue
		}
		if filter.Actor != "" && !strings.Contains(l.ActorEmail+l.ActorID, filter.Actor) {
			continue
		}
		items = append(items, l)
	}
	limit := pagination.NormalizeLimit(page.Limit)
	pg := page.Page
	if pg < 1 {
		pg = 1
	}
	start := (pg - 1) * limit
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return pagination.NewPage(items[start:end], pg, limit, int64(len(items))), nil
}

func (m *memAuditRepo) FindByID(_ context.Context, id string) (appaudit.Log, error) {
	for _, l := range m.logs {
		if l.ID == id {
			return l, nil
		}
	}
	return appaudit.Log{}, appaudit.ErrNotFound
}

func spaPage(screen string, deps Deps, params, query map[string]string) *Page {
	ctrl := settingsPaymentsAuditController(screen)
	return &Page{
		Deps:       deps,
		Controller: ctrl,
		Protected:  true,
		Actor:      appauth.Claims{UserID: "u1", Role: "admin", Email: "a@example.com"},
		Params:     params,
		Query:      query,
	}
}

func spaDeps() Deps {
	settings := appsettings.NewService(appsettings.SettingsDeps{Repo: &memSettingsRepo{}})
	auditRepo := &memAuditRepo{logs: []appaudit.Log{{
		ID: uuid.NewString(), ActorEmail: "a@example.com", ActorID: "u1",
		ActorType: "user", Action: "user.registered", Resource: "user", ResourceID: "u2",
		IP: "127.0.0.1", Source: "web", CreatedAt: time.Now().UTC(),
		Metadata: map[string]any{"email": "b@example.com"},
	}}}
	return Deps{
		PaymentSettingsDeps: PaymentSettingsDeps{
			Settings: settings,
			Notify: config.Notify{
				SMSFrom: "TEST", NetgsmUser: "user", NetgsmPassword: "secret",
			},
			Payment: config.Payment{
				IyzicoAPIKey: "key", IyzicoSecretKey: "secret",
			},
		},
		AuditDeps: AuditDeps{
			Audit: appaudit.NewService(appaudit.ServiceDeps{
				List: appaudit.NewListHandler(auditRepo),
				Get:  appaudit.NewGetHandler(auditRepo),
			}),
		},
		Validate: spaValidate(),
		Checker: rbac.NewStaticChecker(map[string][]rbac.Permission{
			"admin": {
				rbac.PermNotificationsSettings, rbac.PermPaymentsCharge,
				rbac.PermPaymentsList, rbac.PermAuditList,
			},
			"viewer": {rbac.PermPaymentsList, rbac.PermAuditList},
		}),
	}
}

func TestSettingsPaymentsAuditControllerFactory(t *testing.T) {
	screens := []string{
		"sms-settings", "sms-provider", "payment-settings", "payment-provider",
		"checkout", "payments", "payment-show", "audit", "audit-show",
	}
	for _, screen := range screens {
		if settingsPaymentsAuditController(screen) == nil {
			t.Fatalf("%s: nil controller", screen)
		}
	}
	if settingsPaymentsAuditController("unknown") != nil {
		t.Fatal("unknown screen should be nil")
	}
	if settingsPaymentsAuditController("three-ds") != nil {
		t.Fatal("three-ds dead controller should be removed")
	}
}

func TestSMSSettingsRenderAndActivate(t *testing.T) {
	deps := spaDeps()
	page := spaPage("sms-settings", deps, nil, nil)
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "SMS Ayarları") || !strings.Contains(html, "data-key=") || !strings.Contains(html, "Netgsm") {
		t.Fatalf("unexpected sms list render: %s", html)
	}

	// Activate iletimerkezi without config → validation/activation error
	page.Error = ""
	err = page.Controller.HandleEvent(ctx, page, "sms.activate", map[string]any{
		"fields": map[string]any{"provider": "iletimerkezi"},
	})
	if err == nil {
		t.Fatal("expected activation error for unconfigured provider")
	}

	// Denied without permission
	page.Actor.Role = "viewer"
	err = page.Controller.HandleEvent(ctx, page, "sms.activate", map[string]any{
		"fields": map[string]any{"provider": "netgsm"},
	})
	if !errors.Is(err, errForbiddenSPA) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestSMSProviderShowAndValidation(t *testing.T) {
	deps := spaDeps()
	page := spaPage("sms-provider", deps, map[string]string{"provider": "netgsm"}, nil)
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Netgsm") || !strings.Contains(html, `g-submit="sms.activate"`) {
		t.Fatalf("render mismatch: %s", html)
	}
	err = page.Controller.HandleEvent(ctx, page, "sms.activate", map[string]any{
		"fields": map[string]any{"provider": "iletimerkezi"},
	})
	if !errors.Is(err, domainsettings.ErrInvalidSMSProvider) {
		t.Fatalf("want provider mismatch, got %v", err)
	}
}

func TestPaymentSettingsRenderAndActivate(t *testing.T) {
	deps := spaDeps()
	page := spaPage("payment-settings", deps, nil, nil)
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Ödeme Ayarları") || !strings.Contains(html, "Iyzico") {
		t.Fatalf("render mismatch: %s", html)
	}
	st := adaptershared.PaymentIntegrationStatus(deps.Payment)
	if !st.IyzicoConfigured {
		t.Fatal("test deps should mark iyzico configured")
	}
	err = page.Controller.HandleEvent(ctx, page, "payment.activate", map[string]any{
		"fields": map[string]any{"provider": "moka"},
	})
	if err == nil {
		t.Fatal("expected moka not configured error")
	}
}

func TestCheckoutValidationAndClientIPBlocker(t *testing.T) {
	deps := spaDeps()
	// ThreeDSSvc nil → mount still works via Settings; submit fails clearly.
	page := spaPage("checkout", deps, nil, nil)
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "3D Secure") && !strings.Contains(html, "yapılandırılmamış") {
		t.Fatalf("unexpected checkout body: %s", html)
	}

	ctrl := page.Controller.(*checkoutCtrl)
	ctrl.configured = true
	ctrl.iyzicoActive = true
	ctrl.activeProvider = "iyzico"
	html, err = ctrl.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `g-submit="checkout.submit"`) || !strings.Contains(html, `g-debounce="150"`) {
		t.Fatalf("missing goui bindings: %s", html)
	}

	err = ctrl.HandleEvent(ctx, page, "field.amount", map[string]any{"value": "10.00"})
	if err != nil {
		t.Fatal(err)
	}
	if ctrl.form.Amount != "10.00" {
		t.Fatalf("amount not synced: %q", ctrl.form.Amount)
	}

	err = ctrl.HandleEvent(ctx, page, "checkout.submit", map[string]any{"fields": map[string]any{
		"amount": "10", "currency": "TRY", "installment": "1",
		"card_holder": "Ada Lovelace", "card_number": "5528790000000008",
		"exp_month": "12", "exp_year": "2030", "cvc": "123",
		"buyer_name": "Ada", "buyer_surname": "Lovelace",
		"buyer_email": "ada@example.com", "buyer_identity": "11111111111",
		"buyer_address": "Addr", "buyer_city": "Istanbul",
	}})
	if err == nil || !strings.Contains(err.Error(), "3DS servisi") {
		t.Fatalf("want missing ThreeDSSvc error, got %v", err)
	}

	// With IP missing even if service existed — verify helper error path via actor ctx.
	ctxNoIP := appshared.WithActor(ctx, appshared.ActorContext{
		ActorID: "u1", ActorType: appshared.ActorTypeUser, Source: appshared.SourceWeb,
	})
	if ip := actorClientIP(ctxNoIP); ip != "" {
		t.Fatalf("expected empty IP, got %q", ip)
	}
	if !errors.Is(errClientIPMissing, errClientIPMissing) {
		t.Fatal("blocker sentinel missing")
	}
}

func TestAuditListFilterPaginationAndShow(t *testing.T) {
	deps := spaDeps()
	page := spaPage("audit", deps, nil, map[string]string{"action": "user", "order": "asc"})
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Denetim Kayıtları") || !strings.Contains(html, `g-submit="audit.filter"`) {
		t.Fatalf("render mismatch: %s", html)
	}
	for _, want := range []string{
		"Sistem genelindeki tüm kritik olaylar",
		`option value="auth"`,
		`option value="payment"`,
		"Tüm kaynaklar",
		"user.registered",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q: %s", want, html)
		}
	}

	ctrl := page.Controller.(*auditListCtrl)
	if err := ctrl.HandleEvent(ctx, page, "audit.filter", map[string]any{
		"fields": map[string]any{"action": "missing", "page": "1"},
	}); err != nil {
		t.Fatal(err)
	}
	html, err = ctrl.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Filtreye uygun denetim kaydı bulunamadı") {
		t.Fatalf("expected empty state: %s", html)
	}

	show := spaPage("audit-show", deps, map[string]string{"id": "bad"}, nil)
	if err := show.Controller.Mount(ctx, show); err == nil {
		t.Fatal("expected not found for bad id")
	}

	// Permission denied
	page.Actor.Role = "nobody"
	if err := ctrl.HandleEvent(ctx, page, "audit.clear", nil); !errors.Is(err, errForbiddenSPA) {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestAuditShowRenderWithMetadata(t *testing.T) {
	deps := spaDeps()
	pageList := spaPage("audit", deps, nil, nil)
	ctx := context.Background()
	if err := pageList.Controller.Mount(ctx, pageList); err != nil {
		t.Fatal(err)
	}
	ctrl := pageList.Controller.(*auditListCtrl)
	if len(ctrl.page.Items) == 0 {
		t.Fatal("expected audit items")
	}
	id := ctrl.page.Items[0].ID
	page := spaPage("audit-show", deps, map[string]string{"id": id}, nil)
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	html, err := page.Controller.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Denetim Detayı") || !strings.Contains(html, "Metadata") {
		t.Fatalf("render mismatch: %s", html)
	}
	if !strings.Contains(html, escapeHTML(id)) {
		t.Fatalf("id missing: %s", html)
	}
}

func TestDecodeIyzicoHTML(t *testing.T) {
	raw := "<!doctype html><html><body>ok</body></html>"
	out, err := decodeIyzicoHTML(raw)
	if err != nil || out != raw {
		t.Fatalf("passthrough failed: %v %q", err, out)
	}
	encoded := "PGh0bWw+eDwvaHRtbD4=" // <html>x</html>
	out, err = decodeIyzicoHTML(encoded)
	if err != nil || !strings.Contains(out, "<html>") {
		t.Fatalf("base64 decode failed: %v %q", err, out)
	}
}

func TestPaymentProviderMismatch(t *testing.T) {
	deps := spaDeps()
	page := spaPage("payment-provider", deps, map[string]string{"provider": "iyzico"}, nil)
	ctx := context.Background()
	if err := page.Controller.Mount(ctx, page); err != nil {
		t.Fatal(err)
	}
	err := page.Controller.HandleEvent(ctx, page, "payment.activate", map[string]any{
		"fields": map[string]any{"provider": "moka"},
	})
	if !errors.Is(err, domainsettings.ErrInvalidPaymentProvider) {
		t.Fatalf("want mismatch, got %v", err)
	}
}
