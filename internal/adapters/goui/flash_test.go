package goui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestFlash_cookieRoundTrip(t *testing.T) {
	t.Parallel()

	app := fiber.New()
	app.Get("/set", func(c fiber.Ctx) error {
		Flash(c, "success", "kullanıcı başarıyla kaydedildi")
		return c.SendStatus(fiber.StatusNoContent)
	})
	app.Get("/get", func(c fiber.Ctx) error {
		kind, msg := ConsumeFlash(c)
		return c.SendString(kind + "|" + msg)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/set", nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	req := httptest.NewRequest(http.MethodGet, "/get", nil)
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}
	resp2, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	body, _ := io.ReadAll(resp2.Body)
	if got := string(body); got != "success|kullanıcı başarıyla kaydedildi" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyFlash(t *testing.T) {
	t.Parallel()

	p := &Page{}
	applyFlash(p, "success", "ok")
	if p.Notice != "ok" || p.NoticeKind != "success" || p.Error != "" {
		t.Fatalf("success flash: %+v", p)
	}

	p = &Page{}
	applyFlash(p, "error", "fail")
	if p.Error != "fail" || p.Notice != "" {
		t.Fatalf("error flash: %+v", p)
	}

	p = &Page{}
	applyFlash(p, "warning", "careful")
	if p.Notice != "careful" || p.NoticeKind != "warning" {
		t.Fatalf("warning flash: %+v", p)
	}
}

func TestPageRenderFlashMarkers(t *testing.T) {
	t.Parallel()

	page := &Page{
		Controller: &staticCtrl{html: `<p>body</p>`},
		Notice:     "kaydedildi",
		NoticeKind: "success",
		Error:      "hata",
	}
	html, err := page.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-app-flash="success"`,
		`data-app-flash-msg="kaydedildi"`,
		`data-app-flash="error"`,
		`data-app-flash-msg="hata"`,
		`hidden`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, `class="alert`) {
		t.Fatalf("inline alert should not render: %s", html)
	}
}

type staticCtrl struct{ html string }

func (c *staticCtrl) Mount(context.Context, *Page) error { return nil }
func (c *staticCtrl) Render(*Page) (string, error)       { return c.html, nil }
func (c *staticCtrl) HandleEvent(context.Context, *Page, string, map[string]any) error {
	return nil
}
