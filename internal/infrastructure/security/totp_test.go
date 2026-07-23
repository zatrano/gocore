package security

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestTOTP_GenerateAndValidate(t *testing.T) {
	tp := NewTOTP("enterprise")
	secret, uri, err := tp.Generate("user@example.com")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if secret == "" || uri == "" {
		t.Fatal("secret/uri boş olmamalı")
	}

	// Geçerli anlık kod doğrulanmalı.
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	step, ok := tp.Validate(secret, code)
	if !ok || step <= 0 {
		t.Fatal("geçerli kod doğrulanmadı")
	}

	// Yanlış kod reddedilmeli.
	if _, ok := tp.Validate(secret, "000000"); ok {
		t.Fatal("yanlış kod kabul edildi")
	}
}
