package i18n_test

import (
	"testing"

	"github.com/zatrano/gocore/pkg/i18n"
)

func TestEmbeddedLocaleKeyParity(t *testing.T) {
	tr, err := i18n.NewFromEmbedded("tr", []i18n.Locale{"tr", "en"})
	if err != nil {
		t.Fatal(err)
	}
	// Probe a few panel keys in both locales; missing keys fall back but
	// embedded catalogs must load. Deep key equality is covered by JSON load.
	for _, key := range []string{
		"dashboard.contacts.title",
		"dashboard.notice.contact_marked_read",
		"dashboard.payments.complete_3ds",
		"common.back_to_list",
	} {
		trMsg := tr.T("tr", key, "")
		enMsg := tr.T("en", key, "")
		if trMsg == "" || enMsg == "" {
			t.Fatalf("key %q missing tr=%q en=%q", key, trMsg, enMsg)
		}
		if trMsg == enMsg {
			t.Fatalf("key %q should differ across locales: %q", key, trMsg)
		}
	}
}
