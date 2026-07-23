package validation_test

import (
	"testing"

	"github.com/zatrano/gocore/pkg/validation"
)

func TestNormalizeEmail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{"", "", false},
		{"  User@Example.COM ", "user@example.com", false},
		{"not-an-email", "", true},
	}
	for _, tt := range tests {
		got, err := validation.NormalizeEmail(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%q: hata bekleniyordu", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("%q → %q, beklenen %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{"", "", false},
		{"05551112233", "+905551112233", false},
		{"abc", "", true},
	}
	for _, tt := range tests {
		got, err := validation.NormalizePhone(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("%q: hata bekleniyordu", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%q: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("%q → %q, beklenen %q", tt.in, got, tt.want)
		}
	}
}

func TestSanitizeStruct(t *testing.T) {
	type form struct {
		Email string `json:"email" sanitize:"email"`
		Phone string `json:"phone" sanitize:"phone"`
	}
	f := form{Email: "  A@B.COM ", Phone: "05551112233"}
	if err := validation.SanitizeStruct(&f); err != nil {
		t.Fatal(err)
	}
	if f.Email != "a@b.com" || f.Phone != "+905551112233" {
		t.Fatalf("sanitize: %+v", f)
	}
}
