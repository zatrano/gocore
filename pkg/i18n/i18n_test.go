package i18n

import (
	"context"
	"testing"
	"testing/fstest"
)

func newTestTranslator(t *testing.T) *Translator {
	t.Helper()
	fsys := fstest.MapFS{
		"locales/tr.json": &fstest.MapFile{Data: []byte(`{"greeting":"merhaba","items":"{0} adet"}`)},
		"locales/en.json": &fstest.MapFile{Data: []byte(`{"greeting":"hello"}`)},
	}
	tr, err := New(fsys, "locales", "tr", []Locale{"tr", "en"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return tr
}

func TestTranslator_T(t *testing.T) {
	tr := newTestTranslator(t)

	if got := tr.T("en", "greeting", "fb"); got != "hello" {
		t.Errorf("en greeting = %q, want %q", got, "hello")
	}
	// en'de yok → varsayılan dile (tr) düşer.
	if got := tr.T("en", "items", "fb", "3"); got != "3 adet" {
		t.Errorf("fallback to default = %q, want %q", got, "3 adet")
	}
	// hiçbir dilde yok → fallback.
	if got := tr.T("en", "missing", "fallback-text"); got != "fallback-text" {
		t.Errorf("missing = %q, want fallback", got)
	}
	// args ile fallback biçimlenir.
	if got := tr.T("en", "missing", "{0} x", "5"); got != "5 x" {
		t.Errorf("fallback format = %q, want %q", got, "5 x")
	}
}

func TestNew_DefaultMustBeSupported(t *testing.T) {
	fsys := fstest.MapFS{
		"locales/en.json": &fstest.MapFile{Data: []byte(`{}`)},
	}
	if _, err := New(fsys, "locales", "tr", []Locale{"en"}); err == nil {
		t.Fatal("varsayılan dil desteklenmiyorken hata beklenirdi")
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	supported := []Locale{"tr", "en"}
	cases := []struct {
		header string
		want   Locale
	}{
		{"", "tr"},
		{"en", "en"},
		{"en-US,en;q=0.9", "en"},
		{"fr-FR,fr;q=0.9,en;q=0.8", "en"},
		{"de,tr;q=0.5", "tr"},
		{"tr;q=0.3,en;q=0.9", "en"},
		{"*", "tr"},
		{"xx", "tr"},
		{"en;q=0", "tr"},
	}
	for _, tc := range cases {
		if got := ParseAcceptLanguage(tc.header, supported, "tr"); got != tc.want {
			t.Errorf("ParseAcceptLanguage(%q) = %q, want %q", tc.header, got, tc.want)
		}
	}
}

func TestResolve_ExplicitWins(t *testing.T) {
	tr := newTestTranslator(t)
	if got := tr.Resolve("en", "tr,tr;q=0.9"); got != "en" {
		t.Errorf("explicit lang = %q, want en", got)
	}
	if got := tr.Resolve("EN-us", ""); got != "en" {
		t.Errorf("explicit region lang = %q, want en", got)
	}
	// desteklenmeyen explicit → Accept-Language'a düşer.
	if got := tr.Resolve("de", "en"); got != "en" {
		t.Errorf("unsupported explicit = %q, want en (from accept)", got)
	}
}

func TestContext_T(t *testing.T) {
	tr := newTestTranslator(t)
	ctx := NewContext(context.Background(), tr, "en")

	if got := LocaleFromContext(ctx); got != "en" {
		t.Errorf("LocaleFromContext = %q, want en", got)
	}
	if got := T(ctx, "greeting", "fb"); got != "hello" {
		t.Errorf("ctx T greeting = %q, want hello", got)
	}
	// çevirmensiz context → fallback.
	if got := T(context.Background(), "greeting", "fb"); got != "fb" {
		t.Errorf("no-translator T = %q, want fb", got)
	}
}

func TestNewFromEmbedded(t *testing.T) {
	tr, err := NewFromEmbedded(Default, []Locale{"tr", "en"})
	if err != nil {
		t.Fatalf("NewFromEmbedded: %v", err)
	}
	if got := tr.T("en", "user.not_found", "fb"); got != "user not found" {
		t.Errorf("embedded en user.not_found = %q", got)
	}
	if got := tr.T("tr", "user.not_found", "fb"); got != "kullanıcı bulunamadı" {
		t.Errorf("embedded tr user.not_found = %q", got)
	}
}
