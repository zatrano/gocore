package security_test

import (
	"context"
	"testing"

	"github.com/zatrano/gocore/internal/infrastructure/security"
)

func TestArgon2_HashAndVerify(t *testing.T) {
	t.Parallel()
	h := security.NewArgon2Hasher(security.DefaultArgon2Params())
	ctx := context.Background()

	encoded, err := h.Hash(ctx, "s3cret-password")
	if err != nil {
		t.Fatalf("hash hatası: %v", err)
	}

	ok, err := h.Verify(ctx, "s3cret-password", encoded)
	if err != nil || !ok {
		t.Errorf("doğru şifre doğrulanamadı (ok=%v, err=%v)", ok, err)
	}

	ok, _ = h.Verify(ctx, "wrong-password", encoded)
	if ok {
		t.Error("yanlış şifre kabul edildi")
	}
}

func TestArgon2_UniqueSalts(t *testing.T) {
	t.Parallel()
	h := security.NewArgon2Hasher(security.DefaultArgon2Params())
	ctx := context.Background()

	a, _ := h.Hash(ctx, "same")
	b, _ := h.Hash(ctx, "same")
	if a == b {
		t.Error("aynı şifre için hash'ler farklı olmalı (rastgele salt)")
	}
}

func BenchmarkArgon2Hash(b *testing.B) {
	// Not: Argon2 kasıtlı olarak yavaştır; bu benchmark maliyet ayarı içindir.
	h := security.NewArgon2Hasher(security.DefaultArgon2Params())
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = h.Hash(ctx, "benchmark-password")
	}
}
