package goui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/application/authz"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

type memStorage struct {
	objects map[string]memObject
}

type memObject struct {
	data        []byte
	contentType string
}

func newMemStorage() *memStorage {
	return &memStorage{objects: map[string]memObject{}}
}

func (m *memStorage) Put(_ context.Context, key string, r io.Reader, contentType string, size int64) (appshared.FileObject, error) {
	buf, err := io.ReadAll(io.LimitReader(r, size+1))
	if err != nil {
		return appshared.FileObject{}, err
	}
	m.objects[key] = memObject{data: buf, contentType: contentType}
	return appshared.FileObject{Key: key, ContentType: contentType, Size: int64(len(buf))}, nil
}

func (m *memStorage) Get(_ context.Context, key string) (io.ReadCloser, appshared.FileObject, error) {
	obj, ok := m.objects[key]
	if !ok {
		return nil, appshared.FileObject{}, errors.New("not found")
	}
	return io.NopCloser(bytes.NewReader(obj.data)), appshared.FileObject{
		Key: key, ContentType: obj.contentType, Size: int64(len(obj.data)),
	}, nil
}

func (m *memStorage) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func testValidate() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	validation.Register(v)
	return v
}

func pageWithPerms(perms ...rbac.Permission) *Page {
	role := "admin"
	return &Page{
		Deps: Deps{
			Checker:  rbac.NewStaticChecker(map[string][]rbac.Permission{role: perms}),
			Validate: testValidate(),
			Locales:  []string{"tr", "en"},
		},
		Protected: true,
		Actor:     appauth.Claims{Role: role, UserID: "u1", Email: "a@b.com"},
		Params:    map[string]string{},
		Query:     map[string]string{},
	}
}

func TestRbacNotificationsController_Factory(t *testing.T) {
	t.Parallel()
	screens := []string{
		"roles", "role-new", "role-show", "permissions",
		"notification-send", "notification-bulk", "notification-upload", "uploads",
	}
	for _, screen := range screens {
		if rbacNotificationsController(screen) == nil {
			t.Fatalf("screen %q: expected controller", screen)
		}
	}
	if rbacNotificationsController("unknown") != nil {
		t.Fatal("unknown screen should return nil")
	}
}

func TestRolesController_RenderEscapesHTML(t *testing.T) {
	t.Parallel()
	c := &rolesController{roles: []authz.RoleInfo{{
		Name:        `<img src=x onerror=alert(1)>`,
		Description: `A & B <script>`,
		Permissions: []string{"users:list"},
	}}}
	htmlOut, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(htmlOut, "<script>") || strings.Contains(htmlOut, "<img src=") {
		t.Fatalf("unescaped HTML in render: %s", htmlOut)
	}
	if !strings.Contains(htmlOut, "&lt;img") || !strings.Contains(htmlOut, "A &amp; B") || !strings.Contains(htmlOut, "&lt;script&gt;") {
		t.Fatalf("expected escaped content, got %s", htmlOut)
	}
	if !strings.Contains(htmlOut, `data-key="`) {
		t.Fatal("expected data-key on rows")
	}
}

func TestRoleNewController_RenderAndValidation(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermRBACManage)
	c := &roleNewController{
		name:        `evil<script>`,
		description: `desc & more`,
		selected:    map[string]bool{},
	}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(htmlOut, "<script>") {
		t.Fatal("script not escaped")
	}
	if !strings.Contains(htmlOut, `g-submit="role.create"`) {
		t.Fatal("missing g-submit")
	}
	if !strings.Contains(htmlOut, `g-click="role.select_all"`) {
		t.Fatal("missing select_all")
	}

	if err := c.HandleEvent(context.Background(), p, "role.create", map[string]any{
		"fields": map[string]any{"name": "x", "description": ""},
	}); err != nil {
		t.Fatal(err)
	}
	if p.Error == "" {
		t.Fatal("expected validation error for short name")
	}
	if c.fieldErrs == nil {
		t.Fatal("expected field errors")
	}
}

func TestRoleShowController_PermissionDeniedOnEvent(t *testing.T) {
	t.Parallel()
	p := pageWithPerms()
	c := &roleShowController{}
	c.role.Name = "user"
	err := c.HandleEvent(context.Background(), p, "role.update", map[string]any{
		"fields": map[string]any{"description": "x"},
	})
	if !errors.Is(err, errForbiddenAction) {
		t.Fatalf("got %v", err)
	}
}

func TestPermissionsController_RenderMapping(t *testing.T) {
	t.Parallel()
	c := &permissionsController{
		permissions: []authz.PermissionInfo{{Name: "users:list", Description: `A&B<script>`}},
		formName:    `mod:<script>`,
		formDesc:    `A&B`,
	}
	htmlOut, err := c.Render(&Page{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(htmlOut, "<script>") {
		t.Fatal("unescaped")
	}
	if !strings.Contains(htmlOut, `g-submit="permission.create"`) {
		t.Fatal("missing create submit")
	}
	if !strings.Contains(htmlOut, `g-submit="permission.update"`) {
		t.Fatal("missing update submit")
	}
	if !strings.Contains(htmlOut, `data-key="users:list"`) {
		t.Fatal("missing data-key")
	}
}

func TestNotificationSendController_ChannelChangeAndRender(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermNotificationsSend)
	c := &notificationSendController{}
	if err := c.Mount(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `g-submit="notification.send"`) {
		t.Fatal("missing submit")
	}
	if !strings.Contains(htmlOut, `g-change="notification.channel"`) {
		t.Fatal("missing channel change")
	}
	if !strings.Contains(htmlOut, `g-debounce="200"`) {
		t.Fatal("expected debounce on body for inapp")
	}

	if err := c.HandleEvent(context.Background(), p, "notification.channel", map[string]any{"value": "sms"}); err != nil {
		t.Fatal(err)
	}
	if c.channel != "sms" {
		t.Fatalf("channel=%s", c.channel)
	}
	htmlOut, err = c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `name="phone"`) {
		t.Fatal("expected phone field for sms")
	}
	if strings.Contains(htmlOut, `name="email"`) {
		t.Fatal("email should hide for sms")
	}

	p.Error = ""
	_ = c.HandleEvent(context.Background(), p, "notification.send", map[string]any{
		"fields": map[string]any{
			"channel": "sms", "audience": "one", "phone": "bad", "body": "",
		},
	})
	if p.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestNotificationSendController_Forbidden(t *testing.T) {
	t.Parallel()
	p := pageWithPerms()
	c := &notificationSendController{channel: "inapp", audience: "one"}
	err := c.HandleEvent(context.Background(), p, "notification.send", map[string]any{
		"fields": map[string]any{"channel": "inapp", "body": "x", "title": "t", "email": "a@b.com"},
	})
	if !errors.Is(err, errForbiddenAction) {
		t.Fatalf("got %v", err)
	}
}

func TestNotificationBulkController_RenderAndChannel(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermNotificationsSend)
	c := &notificationBulkController{channel: "email", locale: "tr"}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `g-submit="notification.bulk"`) {
		t.Fatal("missing bulk submit")
	}
	_ = c.HandleEvent(context.Background(), p, "notification.bulk_channel", map[string]any{"value": "sms"})
	if c.channel != "sms" {
		t.Fatal(c.channel)
	}
	htmlOut, _ = c.Render(p)
	if !strings.Contains(htmlOut, "telefon") {
		t.Fatal("expected phone recipients hint")
	}

	p.Error = ""
	_ = c.HandleEvent(context.Background(), p, "notification.bulk", map[string]any{
		"fields": map[string]any{"channel": "sms", "body": "", "recipients": ""},
	})
	if p.Error == "" {
		t.Fatal("expected validation error")
	}
}

func TestParseUploadRefAndMapping(t *testing.T) {
	t.Parallel()
	ref := parseUploadRef(map[string]any{
		"id": "abc", "name": "a.csv", "url": "/goui/files/abc",
		"size": "12", "contentType": "text/csv",
	})
	if ref.ID != "abc" || ref.Name != "a.csv" || ref.Size != 12 || ref.ContentType != "text/csv" {
		t.Fatalf("%+v", ref)
	}
	ref2 := parseUploadRef(map[string]any{"value": "x", "name": "n", "size": float64(9)})
	if ref2.ID != "x" || ref2.Size != 9 {
		t.Fatalf("%+v", ref2)
	}
	list := appendUploadRef(nil, ref, true)
	list = appendUploadRef(list, ref, true)
	if len(list) != 1 {
		t.Fatal("duplicate should be ignored")
	}
	list = removeUploadRef(list, "abc")
	if len(list) != 0 {
		t.Fatal("expected empty")
	}
}

func TestNotificationBulkController_HasUploadZone(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermNotificationsSend)
	c := &notificationBulkController{channel: "inapp", locale: "tr"}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `data-goui-upload="1"`) {
		t.Fatal("bulk sayfasında upload zone olmalı")
	}
	if !strings.Contains(htmlOut, "Dosyadan (CSV / Excel)") {
		t.Fatal("csv/excel bölümü eksik")
	}
}

func TestNotificationUploadController_UploadEvents(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermNotificationsSend)
	c := &notificationUploadController{channel: "inapp", locale: "tr"}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `data-goui-upload="1"`) {
		t.Fatal("missing upload zone")
	}
	if !strings.Contains(htmlOut, `g-click="notification.file.uploaded"`) {
		t.Fatal("missing upload carrier")
	}
	_ = c.HandleEvent(context.Background(), p, "notification.file.uploaded", map[string]any{
		"id": "f1", "name": "contacts.csv", "size": "100", "contentType": "text/csv",
	})
	if len(c.files) != 1 || c.files[0].ID != "f1" {
		t.Fatalf("%+v", c.files)
	}
	_ = c.HandleEvent(context.Background(), p, "notification.file.remove", map[string]any{"value": "f1"})
	if len(c.files) != 0 {
		t.Fatal("expected removed")
	}
	p.Error = ""
	_ = c.HandleEvent(context.Background(), p, "notification.upload_send", map[string]any{
		"fields": map[string]any{"channel": "inapp", "title": "t", "body": "b"},
	})
	if p.Error != errMissingUpload.Error() {
		t.Fatalf("got %q", p.Error)
	}
}

func TestUploadsController_RenderAndSubmitMapping(t *testing.T) {
	t.Parallel()
	store := newMemStorage()
	pngHeader := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	_, _ = store.Put(context.Background(), "tmp1", bytes.NewReader(pngHeader), "image/png", int64(len(pngHeader)))

	uploadSvc := appupload.NewService(store, 1<<20, []string{"image/png", "image/jpeg", "application/pdf"}, nil)
	p := pageWithPerms(rbac.PermUploadsCreate)
	p.Deps.Storage = store
	p.Deps.Upload = uploadSvc
	p.Deps.AllowedMIMEs = []string{"image/png", "image/jpeg", "application/pdf"}
	p.Deps.MaxUpload = 1 << 20

	c := &uploadsController{}
	htmlOut, err := c.Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(htmlOut, `g-submit="uploads.submit"`) {
		t.Fatal("missing submit")
	}
	if !strings.Contains(htmlOut, `data-multiple="1"`) {
		t.Fatal("expected multiple uploads")
	}

	_ = c.HandleEvent(context.Background(), p, "uploads.file.uploaded", map[string]any{
		"id": "tmp1", "name": "pic.png", "size": strconv.Itoa(len(pngHeader)), "contentType": "image/png",
	})
	if len(c.files) != 1 {
		t.Fatal(c.files)
	}

	if err := c.HandleEvent(context.Background(), p, "uploads.submit", nil); err != nil {
		t.Fatal(err)
	}
	if p.Notice == "" || c.result == nil || c.result.Accepted != 1 {
		t.Fatalf("notice=%q result=%+v error=%q", p.Notice, c.result, p.Error)
	}
	if _, ok := store.objects["tmp1"]; ok {
		t.Fatal("temp upload key should be cleaned up")
	}
}

func TestUploadsController_ForbiddenAndMissing(t *testing.T) {
	t.Parallel()
	p := pageWithPerms()
	c := &uploadsController{files: []uploadedRef{{ID: "x", Name: "a.png", Size: 1}}}
	err := c.HandleEvent(context.Background(), p, "uploads.submit", nil)
	if !errors.Is(err, errForbiddenAction) {
		t.Fatalf("got %v", err)
	}

	p2 := pageWithPerms(rbac.PermUploadsCreate)
	c2 := &uploadsController{}
	_ = c2.HandleEvent(context.Background(), p2, "uploads.submit", nil)
	if p2.Error != errMissingUpload.Error() {
		t.Fatalf("got %q", p2.Error)
	}
}

func TestIncomingFromStorage(t *testing.T) {
	t.Parallel()
	store := newMemStorage()
	_, _ = store.Put(context.Background(), "k1", strings.NewReader("hello"), "text/plain", 5)
	files, err := incomingFromStorage(context.Background(), store, []uploadedRef{{ID: "k1", Name: "a.txt", Size: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatal(len(files))
	}
	r, err := files[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	data, _ := io.ReadAll(r)
	if string(data) != "hello" {
		t.Fatalf("%q", data)
	}
}

func TestUserFacingErrorAndFieldErrors(t *testing.T) {
	t.Parallel()
	type sample struct {
		Body string `form:"body" validate:"required"`
	}
	err := validation.Check(testValidate(), &sample{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if userFacingError(err) == "" {
		t.Fatal("empty message")
	}
	if len(fieldErrorsFrom(err)) == 0 {
		t.Fatal("expected field map")
	}
}

func TestRequirePagePerm(t *testing.T) {
	t.Parallel()
	p := pageWithPerms(rbac.PermRBACManage)
	if err := requirePagePerm(context.Background(), p, rbac.PermRBACManage); err != nil {
		t.Fatal(err)
	}
	if err := requirePagePerm(context.Background(), p, rbac.PermUploadsCreate); !errors.Is(err, errForbiddenAction) {
		t.Fatalf("got %v", err)
	}
}
