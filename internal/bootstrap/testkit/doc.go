// Package testkit, bootstrap grafiğini tam kurmadan application use-case'lerini
// test etmek için rehberlik sağlar. Production wiring `wire_*.go` dosyalarında
// kalır; testler yalnızca ihtiyaç duydukları portları fake/mock ile doldurur.
//
// Kalıp:
//   - Use-case'in Deps struct'ını (ör. appauth.LoginDeps) doğrudan doldur.
//   - Ağ, DB veya outbox gibi kenar sistemleri fake adapter ile değiştir.
//   - Handler veya HTTP katmanını test ederken gerekirse aynı Deps'i handler
//     constructor'ına da geçir (handler.AuthDeps ↔ appauth.LoginDeps alanları).
//
// Örnek — LoginHandler birim testi:
//
//	hasher := security.NewArgon2Hasher(security.DefaultArgon2Params())
//	sessions := appauth.NewSessionManager(issuer, tokenStore, publisherFake)
//	h := appauth.NewLoginHandler(appauth.LoginDeps{
//	    Users: usersFake, Hasher: hasher, Sessions: sessions,
//	    Guard: guardFake, TOTP: totpFake, MFARepo: mfaFake,
//	    Cache: cacheFake, Tx: txFake, Publisher: pubFake,
//	})
//	// h.Handle(ctx, cmd) ...
//
// Yeni modül eklerken bootstrap'ta ilgili `wire_*.go` dosyasına bakın; testte
// aynı Deps alan sırasını ve türlerini kopyalayın, somut altyapıyı fake ile
// değiştirin. Ağır test framework'ü gerekmez.
package testkit
