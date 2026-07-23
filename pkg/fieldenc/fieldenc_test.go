package fieldenc_test

import (
	"encoding/base64"
	"testing"

	"github.com/zatrano/gocore/pkg/fieldenc"
)

func TestCipherRoundTrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	c, err := fieldenc.New(key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	if got == "secret" {
		t.Fatal("expected ciphertext")
	}
	plain, err := c.Decrypt(got)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "secret" {
		t.Fatalf("got %q", plain)
	}
}

func TestCipherPassthroughWhenDisabled(t *testing.T) {
	c, err := fieldenc.New("")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Encrypt("plain")
	if err != nil || got != "plain" {
		t.Fatalf("got %q err %v", got, err)
	}
}
