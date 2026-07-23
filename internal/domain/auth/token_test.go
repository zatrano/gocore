package auth_test

import (
	"errors"
	"testing"
	"time"

	domainauth "github.com/zatrano/gocore/internal/domain/auth"
)

func TestNewOneTimeToken(t *testing.T) {
	exp := time.Now().UTC().Add(time.Hour)
	tok, err := domainauth.NewOneTimeToken("user-1", domainauth.PurposeEmailVerify, "abc123", exp)
	if err != nil {
		t.Fatalf("NewOneTimeToken: %v", err)
	}
	if tok.UserID() != "user-1" {
		t.Fatalf("userID = %q", tok.UserID())
	}
}

func TestOneTimeToken_Consume(t *testing.T) {
	exp := time.Now().UTC().Add(time.Hour)
	tok, _ := domainauth.NewOneTimeToken("user-1", domainauth.PurposeEmailVerify, "hash", exp)
	now := time.Now().UTC()

	if err := tok.Consume(domainauth.PurposeEmailVerify, now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := tok.Consume(domainauth.PurposeEmailVerify, now); err == nil {
		t.Fatal("second consume should fail")
	}
}

func TestValidatePasswordLength(t *testing.T) {
	if err := domainauth.ValidatePasswordLength("1234567"); !errors.Is(err, domainauth.ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
	if err := domainauth.ValidatePasswordLength("12345678"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}
