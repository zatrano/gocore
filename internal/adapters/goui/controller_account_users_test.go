package goui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	appauthz "github.com/zatrano/gocore/internal/application/authz"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

func testAccountValidate() *validator.Validate {
	v := validator.New()
	validation.Register(v)
	return v
}

func TestAccountUsersController_Factory(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"dashboard": true,
		"account":   true,
		"mfa":       true,
		"inbox":     true,
		"users":     true,
		"user-new":  true,
		"user-show": true,
		"login":     false,
		"":          false,
		"unknown":   false,
	}
	for screen, want := range cases {
		got := accountUsersController(screen)
		if want && got == nil {
			t.Fatalf("%q: beklenen Controller, nil geldi", screen)
		}
		if !want && got != nil {
			t.Fatalf("%q: beklenen nil, Controller geldi", screen)
		}
	}
}

func TestDisplayErrAndFieldErrors(t *testing.T) {
	t.Parallel()
	de := shared.NewDomainError(shared.KindForbidden, "auth.forbidden", "bu işlem için yetkiniz yok")
	if got := accountDisplayErr(de); got != "bu işlem için yetkiniz yok" {
		t.Fatalf("domain mesajı: %q", got)
	}

	v := testAccountValidate()
	req := accountChangeNameForm{Name: "x"}
	err := validation.Check(v, &req)
	if err == nil {
		t.Fatal("doğrulama hatası beklenirdi")
	}
	if got := accountDisplayErr(err); !strings.Contains(got, "geçersiz") {
		t.Fatalf("validation display: %q", got)
	}
	fields := accountFieldErrors(context.Background(), err)
	if fields["name"] == "" {
		t.Fatalf("alan hatası yok: %#v", fields)
	}
}

func TestDashboardRender(t *testing.T) {
	t.Parallel()
	checker := rbac.NewStaticChecker(map[string][]rbac.Permission{
		"admin": {rbac.PermUsersList},
	})
	c := &dashboardController{}
	page := &Page{
		Deps:      Deps{Checker: checker},
		Actor:     appauth.Claims{Role: "admin", Email: `admin<script>@x.com`},
		Protected: true,
	}
	html, err := c.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Dashboard") {
		t.Fatal("başlık yok")
	}
	if !strings.Contains(html, "admin&lt;script&gt;@x.com") {
		t.Fatalf("e-posta escape edilmedi: %s", html)
	}
	if !strings.Contains(html, `/dashboard/users`) {
		t.Fatal("kullanıcılar kısayolu yok")
	}
}

func TestAccountRenderForms(t *testing.T) {
	t.Parallel()
	c := &accountController{
		profile: appuser.View{
			Name: "Ali", Email: "ali@example.com", Phone: "+905551112233",
			Role: "user", PreferredLocale: "tr", MFAEnabled: false, EmailVerified: true,
		},
		formName: "Ali", formEmail: "ali@example.com", formPhone: "+905551112233", formLocale: "tr",
	}
	page := &Page{Deps: Deps{Locales: []string{"tr", "en"}}}
	html, err := c.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		`g-submit="account.change_name"`,
		`g-submit="account.change_email"`,
		`g-submit="account.change_phone"`,
		`g-submit="account.change_locale"`,
		`g-submit="account.change_password"`,
		`/dashboard/account/mfa`,
		`/dashboard/account/notifications`,
		`value="Ali"`,
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("eksik: %s", needle)
		}
	}
}

func TestAccountValidationRejectsShortName(t *testing.T) {
	t.Parallel()
	c := &accountController{profile: appuser.View{Name: "Ali", PreferredLocale: "tr"}}
	page := &Page{
		Deps:  Deps{Validate: testAccountValidate(), Locales: []string{"tr"}},
		Actor: appauth.Claims{UserID: "u1", Role: "user"},
	}
	if err := c.HandleEvent(context.Background(), page, "account.change_name", map[string]any{
		"fields": map[string]any{"name": "x"},
	}); err != nil {
		t.Fatal(err)
	}
	if page.Error == "" {
		t.Fatal("doğrulama hatası beklenirdi")
	}
	if c.fieldErrors["name"] == "" {
		t.Fatalf("alan hatası yok: %#v", c.fieldErrors)
	}
	// ChangeName servisi nil; doğrulama erken kesmeli — panic yok.
}

func TestMFARenderStates(t *testing.T) {
	t.Parallel()
	c := &mfaController{profile: appuser.View{MFAEnabled: false}}
	html, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `g-submit="mfa.setup"`) {
		t.Fatal("setup formu yok")
	}

	c.setup = &appauth.SetupResult{
		Secret:    "SEC<script>",
		URI:       "otpauth://x",
		QRDataURI: "data:image/png;base64,abc+/=",
	}
	html, err = c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "SEC&lt;script&gt;") {
		t.Fatal("secret escape edilmedi")
	}
	if !strings.Contains(html, `src="data:image/png;base64,`) {
		t.Fatalf("QR data URI must render in img src, got: %s", html)
	}
	if strings.Contains(html, "ZgotmplZ") {
		t.Fatal("html/template must not sanitize QR data URI")
	}
	if !strings.Contains(html, `g-submit="mfa.enable"`) {
		t.Fatal("enable formu yok")
	}

	c.setup = nil
	c.profile.MFAEnabled = true
	html, err = c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `g-submit="mfa.disable"`) {
		t.Fatal("disable formu yok")
	}
}

func TestMFAEnableValidation(t *testing.T) {
	t.Parallel()
	c := &mfaController{profile: appuser.View{}, setup: &appauth.SetupResult{Secret: "s", URI: "u"}}
	page := &Page{
		Deps: Deps{
			Validate: testAccountValidate(),
			AuthDeps: AuthDeps{Auth: appauth.NewService(appauth.ServiceDeps{MFA: &appauth.MFAHandler{}})},
		},
		Actor: appauth.Claims{UserID: "u1"},
	}
	if err := c.HandleEvent(context.Background(), page, "mfa.enable", map[string]any{
		"fields": map[string]any{"code": ""},
	}); err != nil {
		t.Fatal(err)
	}
	if page.Error == "" || c.fieldErrors["code"] == "" {
		t.Fatalf("kod doğrulaması beklenirdi error=%q fields=%#v", page.Error, c.fieldErrors)
	}
}

func TestInboxRenderMarkReadAndFilter(t *testing.T) {
	t.Parallel()
	now := datetime.FromTime(time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	c := &inboxController{
		unread: 1,
		page: pagination.Page[appnotif.View]{
			Page: 1, Limit: 100, Total: 2, TotalPages: 1,
			Items: []appnotif.View{
				{ID: "n1", Title: "Merhaba<script>", Content: "içerik", Read: false, CreatedAt: now},
				{ID: "n2", Title: "Eski", Content: "okundu", Read: true, CreatedAt: now},
			},
		},
	}
	html, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "Merhaba&lt;script&gt;") {
		t.Fatal("başlık escape edilmedi")
	}
	if !strings.Contains(html, `data-key="n1"`) {
		t.Fatal("data-key yok")
	}
	if !strings.Contains(html, `g-click="inbox.mark_read"`) {
		t.Fatal("mark_read yok")
	}
	if !strings.Contains(html, `g-click="inbox.mark_all_read"`) {
		t.Fatal("mark_all_read yok")
	}
	if !strings.Contains(html, `g-submit="inbox.delete"`) {
		t.Fatal("delete yok")
	}
	if !strings.Contains(html, `g-submit="inbox.delete_all"`) {
		t.Fatal("delete_all yok")
	}
	if !strings.Contains(html, `data-goui-value="unread"`) {
		t.Fatal("okunmamış filtresi yok")
	}

	c.unread = 0
	c.unreadOnly = false
	html, err = c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, `data-goui-value="unread"`) {
		t.Fatal("okunmamış yokken Okunmamış butonu çıkmamalı")
	}
	if strings.Contains(html, `inbox.mark_all_read`) {
		t.Fatal("okunmamış yokken tümünü okundu yap çıkmamalı")
	}

	c.unread = 1
	c.unreadOnly = true
	c.page.Items = []appnotif.View{
		{ID: "n1", Title: "Merhaba<script>", Content: "içerik", Read: false, CreatedAt: now},
	}
	c.page.Total = 1
	html, err = c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, `data-key="n2"`) {
		t.Fatal("okunmuş öğe filtrede görünmemeli")
	}
	if !strings.Contains(html, `data-key="n1"`) {
		t.Fatal("okunmamış öğe kayboldu")
	}
}

func TestInboxFilterEvent(t *testing.T) {
	t.Parallel()
	c := &inboxController{}
	page := &Page{Actor: appauth.Claims{UserID: "u1"}, Query: map[string]string{}}
	// Notifications nil → reload hata yazar; filter state yine set edilmeli.
	_ = c.HandleEvent(context.Background(), page, "inbox.filter", map[string]any{"value": "unread"})
	if !c.unreadOnly {
		t.Fatal("unreadOnly set edilmedi")
	}
	if page.Query["filter"] != "unread" {
		t.Fatalf("query filter=%q", page.Query["filter"])
	}
}

func TestUsersListRenderSearchDebounce(t *testing.T) {
	t.Parallel()
	c := &usersListController{
		page: pagination.Page[appuser.View]{
			Page: 1, Limit: 100, Total: 1, TotalPages: 1,
			Items: []appuser.View{{ID: "u1", Name: "Ada<script>", Email: "a@b.co", Role: "admin", Active: true}},
		},
	}
	page := &Page{Query: map[string]string{"search": "Ada", "role": "admin"}, Actor: appauth.Claims{Role: "admin"}}
	html, err := c.Render(page)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		`g-input="users.search"`,
		`g-debounce="300"`,
		`g-change="users.deleted"`,
		`data-key="u1"`,
		`Ada&lt;script&gt;`,
		`/dashboard/users/u1`,
		`/dashboard/users/new`,
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("eksik: %s\n%s", needle, html)
		}
	}
}

func TestUsersListAccessDeniedOnReload(t *testing.T) {
	t.Parallel()
	checker := rbac.NewStaticChecker(map[string][]rbac.Permission{
		"user": {},
	})
	c := &usersListController{}
	page := &Page{
		Deps: Deps{
			UserDeps: UserDeps{Users: appuser.NewService(appuser.ServiceDeps{Access: appuser.NewAccess(checker)})},
			Checker:  checker,
		},
		Actor: appauth.Claims{UserID: "u1", Role: "user"},
		Query: map[string]string{},
	}
	if err := c.Mount(context.Background(), page); err != nil {
		t.Fatal(err)
	}
	if page.Error == "" || !strings.Contains(page.Error, "yetkiniz yok") {
		t.Fatalf("yetki reddi beklenirdi: %q", page.Error)
	}
}

func TestUserNewValidationAndAccess(t *testing.T) {
	t.Parallel()
	deny := rbac.NewStaticChecker(map[string][]rbac.Permission{"user": {}})
	c := &userNewController{}
	page := &Page{
		Deps: Deps{
			UserDeps: UserDeps{Users: appuser.NewService(appuser.ServiceDeps{Access: appuser.NewAccess(deny)})},
			Validate: testAccountValidate(),
		},
		Actor: appauth.Claims{UserID: "u1", Role: "user"},
	}
	if err := c.HandleEvent(context.Background(), page, "users.create", map[string]any{
		"fields": map[string]any{
			"name": "Ada Lovelace", "email": "ada@example.com",
			"password": "secret123", "role": "user",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if page.Error == "" || !strings.Contains(page.Error, "yetkiniz yok") {
		t.Fatalf("create yetki reddi beklenirdi: %q", page.Error)
	}

	allow := rbac.NewStaticChecker(map[string][]rbac.Permission{
		"admin": {rbac.PermUsersList},
	})
	c = &userNewController{roles: []appauthz.RoleInfo{{Name: "user"}}}
	page = &Page{
		Deps: Deps{
			UserDeps: UserDeps{Users: appuser.NewService(appuser.ServiceDeps{Access: appuser.NewAccess(allow)})},
			Validate: testAccountValidate(),
		},
		Actor: appauth.Claims{UserID: "a1", Role: "admin"},
	}
	if err := c.HandleEvent(context.Background(), page, "users.create", map[string]any{
		"fields": map[string]any{"name": "x", "email": "bad", "password": "short", "role": "user"},
	}); err != nil {
		t.Fatal(err)
	}
	if page.Error == "" {
		t.Fatal("form doğrulaması beklenirdi")
	}
	if len(c.fieldErrors) == 0 {
		t.Fatal("alan hataları beklenirdi")
	}
}

func TestUserNewRenderEscapes(t *testing.T) {
	t.Parallel()
	c := &userNewController{
		roles: []appauthz.RoleInfo{{Name: "user"}},
		form:  adminRegisterForm{Name: `<img>`, Email: `a@b.co`, Role: "user"},
	}
	html, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `g-submit="users.create"`) {
		t.Fatal("create formu yok")
	}
	if !strings.Contains(html, `&lt;img&gt;`) {
		t.Fatal("name escape edilmedi")
	}
}

func TestUserShowRenderPermGates(t *testing.T) {
	t.Parallel()
	c := &userShowController{
		profile: appuser.View{
			ID: "u2", Name: "Bob<script>", Email: "b@e.co", Role: "user",
			Active: false, CreatedAt: datetime.FromTime(time.Now().UTC()),
		},
		perms: userShowPerms{
			CanChangeRole: true, CanActivate: true, CanChangeProfileAny: true, CanDelete: true,
		},
		roles:        []appauthz.RoleInfo{{Name: "user"}, {Name: "admin"}},
		selectedRole: "user",
	}
	html, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		`Bob&lt;script&gt;`,
		`g-submit="user.activate"`,
		`g-submit="user.change_role"`,
		`g-submit="user.change_name"`,
		`g-submit="user.change_email"`,
		`g-submit="user.change_phone"`,
		`g-submit="user.set_password"`,
		`name="new_password"`,
		`name="confirm_password"`,
		`account-settings-grid`,
		`g-submit="user.delete"`,
	} {
		if !strings.Contains(html, needle) {
			t.Fatalf("eksik: %s", needle)
		}
	}

	c.perms = userShowPerms{}
	html, err = c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "Yönetim") {
		t.Fatal("izin yokken yönetim kartı görünmemeli")
	}
}

func TestUserShowPermissionDeniedEvents(t *testing.T) {
	t.Parallel()
	checker := rbac.NewStaticChecker(map[string][]rbac.Permission{
		"user": {},
	})
	c := &userShowController{profile: appuser.View{ID: "u2", Name: "Bob"}}
	page := &Page{
		Deps: Deps{
			UserDeps: UserDeps{Users: appuser.NewService(appuser.ServiceDeps{Access: appuser.NewAccess(checker)})},
			Validate: testAccountValidate(),
		},
		Actor:  appauth.Claims{UserID: "u1", Role: "user"},
		Params: map[string]string{"id": "u2"},
	}

	events := []string{"user.change_role", "user.activate", "user.delete", "user.restore", "user.set_password"}
	for _, ev := range events {
		page.Error = ""
		if err := c.HandleEvent(context.Background(), page, ev, map[string]any{
			"fields": map[string]any{
				"role": "admin", "name": "Bob", "email": "b@e.co", "phone": "",
				"new_password": "password1", "confirm_password": "password1",
			},
		}); err != nil {
			t.Fatalf("%s: %v", ev, err)
		}
		if page.Error == "" || !strings.Contains(page.Error, "yetkiniz yok") {
			t.Fatalf("%s: yetki reddi beklenirdi got=%q", ev, page.Error)
		}
	}
}

func TestUserShowSelfProfileChangeAllowedByAccess(t *testing.T) {
	t.Parallel()
	// users:email:change:any yok; self için CanChangeProfileAny geçer.
	checker := rbac.NewStaticChecker(map[string][]rbac.Permission{"user": {}})
	access := appuser.NewAccess(checker)
	if err := access.CanChangeProfileAny(context.Background(), "user", "u1", "u1"); err != nil {
		t.Fatal("self profil değişimi izinli olmalı")
	}
	if err := access.CanChangeProfileAny(context.Background(), "user", "u1", "u2"); err == nil {
		t.Fatal("başka kullanıcı profili reddedilmeli")
	}
}

func TestPayloadHelpers(t *testing.T) {
	t.Parallel()
	if got := payloadString(map[string]any{"fields": map[string]any{"name": "  Ada  "}}, "name"); got != "Ada" {
		t.Fatalf("payloadString=%q", got)
	}
	if got := payloadID(map[string]any{"id": "n9"}); got != "n9" {
		t.Fatalf("payloadID=%q", got)
	}
	if got := payloadPage(map[string]any{"value": "3"}, 1); got != 3 {
		t.Fatalf("payloadPage=%d", got)
	}
}

func TestPaginationHTML(t *testing.T) {
	t.Parallel()
	html := renderPagination("users", 2, 5)
	if !strings.Contains(html, `g-click="users.page"`) {
		t.Fatal("page click yok")
	}
	if !strings.Contains(html, `data-key="page-2"`) {
		t.Fatal("data-key yok")
	}
	if renderPagination("inbox", 1, 1) != "" {
		t.Fatal("tek sayfada pagination olmamalı")
	}
}
