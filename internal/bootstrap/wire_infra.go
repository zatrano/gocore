package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/zatrano/gocore/internal/infrastructure/cache"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/fieldenc"
	"github.com/zatrano/gocore/pkg/i18n"
)

func (g *graph) wireInfra(ctx context.Context) error {
	translator, err := i18n.NewFromEmbedded(i18n.Locale(g.cfg.I18n.DefaultLocale), toLocales(g.cfg.I18n.Supported))
	if err != nil {
		return fmt.Errorf("bootstrap: i18n: %w", err)
	}
	g.translator = translator

	pool, err := database.NewPool(ctx, g.cfg.DB)
	if err != nil {
		return fmt.Errorf("bootstrap: db: %w", err)
	}
	g.app.pool = pool
	g.txManager = database.NewTxManager(pool)

	mfaEncryptionKey := g.cfg.Auth.MFAEncryptionKey.Value()
	if mfaEncryptionKey == "" && !g.cfg.App.IsProduction() {
		// Geliştirme geçişi: mevcut payment anahtarını kullan. Production,
		// kriptografik anahtar ayrımı için ayrı AUTH_MFA_ENCRYPTION_KEY ister.
		mfaEncryptionKey = g.cfg.Payment.FieldEncryptionKey.Value()
	}
	if mfaEncryptionKey == "" && !g.cfg.App.IsProduction() {
		// Yerel geliştirmede dahi plaintext bırakma; sabit JWT secret'tan
		// alan-özel bir geliştirme anahtarı türet.
		sum := sha256.Sum256([]byte("gocore:mfa-field:" + g.cfg.Auth.JWTSecret.Value()))
		mfaEncryptionKey = base64.StdEncoding.EncodeToString(sum[:])
	}
	mfaFieldCipher, err := fieldenc.New(mfaEncryptionKey)
	if err != nil {
		return fmt.Errorf("bootstrap: MFA alan şifreleme: %w", err)
	}
	g.mfaFieldCipher = mfaFieldCipher

	g.memCache = cache.NewMemory()
	return nil
}
