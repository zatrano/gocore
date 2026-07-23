package audit

import "testing"

func TestMaskEmail(t *testing.T) {
	if got := MaskEmail("alice@example.com"); got != "a***@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskPhone(t *testing.T) {
	if got := MaskPhone("+905551112233"); !stringsHasSuffix(got, "2233") {
		t.Fatalf("got %q", got)
	}
}

func TestRedactMetadata(t *testing.T) {
	in := map[string]any{
		"password": "secret",
		"email":    "bob@example.com",
		"card_pan": "4111111111111111",
		"nested": map[string]any{
			"token": "abc",
			"ok":    1,
		},
	}
	out := RedactMetadata(in)
	if out["password"] != "[redacted]" {
		t.Fatalf("password not redacted: %v", out["password"])
	}
	if out["card_pan"] != "[redacted]" {
		t.Fatalf("card not redacted")
	}
	if out["email"] != "b***@example.com" {
		t.Fatalf("email mask: %v", out["email"])
	}
	nested, _ := out["nested"].(map[string]any)
	if nested["token"] != "[redacted]" {
		t.Fatalf("nested token: %v", nested["token"])
	}
}

func stringsHasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
