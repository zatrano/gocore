package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainauth "github.com/zatrano/gocore/internal/domain/auth"
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/internal/infrastructure/cache"
	"github.com/zatrano/gocore/pkg/pagination"
)

type mfaTestUserRepo struct{ u *user.User }

func (r *mfaTestUserRepo) Save(context.Context, *user.User) error       { return nil }
func (r *mfaTestUserRepo) Update(_ context.Context, u *user.User) error { r.u = u; return nil }
func (r *mfaTestUserRepo) FindByID(context.Context, user.ID) (*user.User, error) {
	return r.u, nil
}
func (r *mfaTestUserRepo) FindByIDIncludeDeleted(context.Context, user.ID) (*user.User, error) {
	return r.u, nil
}
func (r *mfaTestUserRepo) FindByEmail(context.Context, user.Email) (*user.User, error) {
	return r.u, nil
}
func (r *mfaTestUserRepo) FindByIDs(context.Context, []user.ID) ([]*user.User, error) {
	return []*user.User{r.u}, nil
}
func (r *mfaTestUserRepo) List(context.Context, user.ListFilter, pagination.Request) (pagination.Page[*user.User], error) {
	return pagination.NewPage([]*user.User{r.u}, 1, 20, 1), nil
}
func (r *mfaTestUserRepo) ExistsByEmail(context.Context, user.Email) (bool, error) {
	return true, nil
}
func (r *mfaTestUserRepo) Delete(context.Context, user.ID) error     { return nil }
func (r *mfaTestUserRepo) Restore(context.Context, user.ID) error    { return nil }
func (r *mfaTestUserRepo) HardDelete(context.Context, user.ID) error { return nil }
func (r *mfaTestUserRepo) CountActiveByRole(context.Context, user.Role) (int64, error) {
	return 1, nil
}

type mfaTestRepo struct {
	lastStep int64
	hashes   map[string]bool
}

func (r *mfaTestRepo) ConsumeTOTPStep(_ context.Context, _ string, step int64) (bool, error) {
	if step <= r.lastStep {
		return false, nil
	}
	r.lastStep = step
	return true, nil
}
func (r *mfaTestRepo) ResetTOTPStep(context.Context, string) error {
	r.lastStep = -1
	return nil
}
func (r *mfaTestRepo) ReplaceRecoveryCodes(_ context.Context, _ string, hashes []string) error {
	r.hashes = make(map[string]bool, len(hashes))
	for _, hash := range hashes {
		r.hashes[hash] = true
	}
	return nil
}
func (r *mfaTestRepo) ConsumeRecoveryCode(_ context.Context, _, hash string) (bool, error) {
	if !r.hashes[hash] {
		return false, nil
	}
	delete(r.hashes, hash)
	return true, nil
}
func (r *mfaTestRepo) DeleteRecoveryCodes(context.Context, string) error {
	r.hashes = nil
	return nil
}

type mfaTestTOTP struct{}

func (mfaTestTOTP) Generate(string) (string, string, error) {
	return "JBSWY3DPEHPK3PXP",
		"otpauth://totp/GoCore:user@example.com?issuer=GoCore&secret=JBSWY3DPEHPK3PXP", nil
}
func (mfaTestTOTP) Validate(_ string, code string) (int64, bool) {
	return 100, code == "123456"
}

type mfaTestHasher struct{}

func (mfaTestHasher) Hash(context.Context, string) (string, error) { return "hash", nil }
func (mfaTestHasher) Verify(_ context.Context, plain, _ string) (bool, error) {
	return plain == "password", nil
}
func (mfaTestHasher) NeedsRehash(string) bool { return false }

type mfaTestGuard struct{}

func (mfaTestGuard) Allowed(context.Context, string) (bool, error) { return true, nil }
func (mfaTestGuard) RecordFailure(context.Context, string) error   { return nil }
func (mfaTestGuard) Reset(context.Context, string) error           { return nil }

type mfaTestTx struct{}

func (mfaTestTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mfaTestIssuer struct{ claims map[string]Claims }

func (i *mfaTestIssuer) Issue(_ context.Context, c Claims) (TokenPair, error) {
	if i.claims == nil {
		i.claims = map[string]Claims{}
	}
	now := time.Now().UTC()
	c.TokenID, c.IssuedAt, c.ExpiresAt = "access-id", now, now.Add(time.Hour)
	access := "access"
	i.claims[access] = c
	rc := c
	rc.TokenID, rc.Type = "refresh-id", "refresh"
	refresh := "refresh"
	i.claims[refresh] = rc
	return TokenPair{AccessToken: access, RefreshToken: refresh, ExpiresAt: c.ExpiresAt}, nil
}
func (i *mfaTestIssuer) Verify(_ context.Context, raw string) (Claims, error) {
	c, ok := i.claims[raw]
	if !ok {
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}
func (i *mfaTestIssuer) Inspect(ctx context.Context, raw string) (Claims, error) {
	return i.Verify(ctx, raw)
}

type mfaTestTokenStore struct{ revokedAt time.Time }

func (s *mfaTestTokenStore) ActivateRefresh(context.Context, string, string, time.Time) error {
	return nil
}
func (s *mfaTestTokenStore) IsRefreshActive(context.Context, string, string) (bool, error) {
	return true, nil
}
func (s *mfaTestTokenStore) WasRefreshConsumed(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *mfaTestTokenStore) ConsumeRefresh(context.Context, string, string) error  { return nil }
func (s *mfaTestTokenStore) RevokeAccess(context.Context, string, time.Time) error { return nil }
func (s *mfaTestTokenStore) IsAccessRevoked(context.Context, string) (bool, error) {
	return false, nil
}
func (s *mfaTestTokenStore) RevokeAllForUser(_ context.Context, _ string, at time.Time) error {
	s.revokedAt = at
	return nil
}
func (s *mfaTestTokenStore) UserRevokedAt(context.Context, string) (time.Time, error) {
	return s.revokedAt, nil
}

func mfaTestUser(t *testing.T, enabled bool) *user.User {
	t.Helper()
	email, _ := user.NewEmail("user@example.com")
	phone, _ := user.NewPhone("")
	password, _ := user.NewHashedPassword("hash")
	role, _ := user.ParseRole("user")
	locale, _ := user.ParsePreferredLocale("tr")
	now := time.Now().UTC()
	return user.Hydrate(
		user.NewID(), email, phone, "Test User", password, role,
		true, true, enabled, "JBSWY3DPEHPK3PXP", locale, now, now, nil,
	)
}

func TestLoginMFAChallengeHidesPasswordAndPreventsReplay(t *testing.T) {
	u := mfaTestUser(t, true)
	users := &mfaTestUserRepo{u: u}
	mfaRepo := &mfaTestRepo{lastStep: -1, hashes: map[string]bool{}}
	sessions := NewSessionManager(&mfaTestIssuer{}, &mfaTestTokenStore{}, nil)
	handler := NewLoginHandler(LoginDeps{
		Users: users, Hasher: mfaTestHasher{}, Sessions: sessions, Guard: mfaTestGuard{},
		TOTP: mfaTestTOTP{}, MFARepo: mfaRepo, Cache: cache.NewMemory(), Tx: mfaTestTx{}, Publisher: nil,
	})

	_, err := handler.Handle(context.Background(), LoginCommand{
		Email: "user@example.com", Password: "password", ClientKey: "127.0.0.1",
	})
	challenge, ok := MFAChallengeFrom(err)
	if !ok || challenge == "" {
		t.Fatalf("MFA challenge beklenirdi: %v", err)
	}

	tokens, err := handler.Handle(context.Background(), LoginCommand{
		MFAChallenge: challenge, MFACode: "123456", ClientKey: "127.0.0.1",
	})
	if err != nil || tokens.AccessToken == "" {
		t.Fatalf("MFA tamamlama başarısız: tokens=%+v err=%v", tokens, err)
	}

	_, err = handler.Handle(context.Background(), LoginCommand{
		MFAChallenge: challenge, MFACode: "123456", ClientKey: "127.0.0.1",
	})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("challenge tekrar kullanımı reddedilmeliydi: %v", err)
	}
}

func TestLoginRecoveryCodeIsSingleUse(t *testing.T) {
	u := mfaTestUser(t, true)
	users := &mfaTestUserRepo{u: u}
	recovery := "ABCD-EFGH-JKLM-NPQR"
	mfaRepo := &mfaTestRepo{
		lastStep: -1,
		hashes: map[string]bool{
			domainauth.HashToken(normalizeRecoveryCode(recovery)): true,
		},
	}
	handler := NewLoginHandler(LoginDeps{
		Users: users, Hasher: mfaTestHasher{},
		Sessions: NewSessionManager(&mfaTestIssuer{}, &mfaTestTokenStore{}, nil),
		Guard:    mfaTestGuard{}, TOTP: mfaTestTOTP{}, MFARepo: mfaRepo,
		Cache: cache.NewMemory(), Tx: mfaTestTx{}, Publisher: nil,
	})

	begin := func() string {
		t.Helper()
		_, err := handler.Handle(context.Background(), LoginCommand{
			Email: "user@example.com", Password: "password", ClientKey: "127.0.0.1",
		})
		challenge, ok := MFAChallengeFrom(err)
		if !ok {
			t.Fatalf("challenge beklenirdi: %v", err)
		}
		return challenge
	}

	if _, err := handler.Handle(context.Background(), LoginCommand{
		MFAChallenge: begin(), MFACode: recovery, ClientKey: "127.0.0.1",
	}); err != nil {
		t.Fatalf("recovery code kabul edilmeliydi: %v", err)
	}
	if _, err := handler.Handle(context.Background(), LoginCommand{
		MFAChallenge: begin(), MFACode: recovery, ClientKey: "127.0.0.1",
	}); !errors.Is(err, ErrInvalidMFACode) {
		t.Fatalf("recovery code tekrar kullanımı reddedilmeliydi: %v", err)
	}
}

func TestMFAEnableReturnsOneTimeRecoveryCodesAndRevokesSessions(t *testing.T) {
	u := mfaTestUser(t, false)
	users := &mfaTestUserRepo{u: u}
	mfaRepo := &mfaTestRepo{lastStep: -1}
	store := &mfaTestTokenStore{}
	sessions := NewSessionManager(&mfaTestIssuer{}, store, nil)
	handler := NewMFAHandler(MFADeps{
		Users: users, MFARepo: mfaRepo, TOTP: mfaTestTOTP{}, Sessions: sessions, Pub: nil, Tx: mfaTestTx{},
	})

	setup, err := handler.Setup(context.Background(), u.ID().String())
	if err != nil || setup.Secret == "" || setup.QRDataURI == "" {
		t.Fatalf("setup eksik: %+v err=%v", setup, err)
	}
	result, err := handler.Enable(context.Background(), EnableCommand{
		UserID: u.ID().String(), Code: "123456",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RecoveryCodes) != recoveryCodeCount || len(mfaRepo.hashes) != recoveryCodeCount {
		t.Fatalf("recovery codes=%d hashes=%d", len(result.RecoveryCodes), len(mfaRepo.hashes))
	}
	if store.revokedAt.IsZero() {
		t.Fatal("MFA etkinleştirme diğer oturumları iptal etmeliydi")
	}
}
