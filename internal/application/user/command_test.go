package user_test

import (
	"context"
	"errors"
	"testing"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	appuser "github.com/zatrano/gocore/internal/application/user"
	dshared "github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/pagination"
)

// --- Elle yazılmış fake'ler (mockery yerine; bağımsız ve hızlı testler) ---

type fakeRepo struct {
	saved      *user.User
	existsResp bool
}

func (f *fakeRepo) Save(_ context.Context, u *user.User) error { f.saved = u; return nil }
func (f *fakeRepo) Update(context.Context, *user.User) error   { return nil }
func (f *fakeRepo) FindByID(context.Context, user.ID) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (f *fakeRepo) FindByIDIncludeDeleted(context.Context, user.ID) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (f *fakeRepo) FindByEmail(context.Context, user.Email) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (f *fakeRepo) FindByIDs(context.Context, []user.ID) ([]*user.User, error) { return nil, nil }
func (f *fakeRepo) List(context.Context, user.ListFilter, pagination.Request) (pagination.Page[*user.User], error) {
	return pagination.Page[*user.User]{}, nil
}
func (f *fakeRepo) ExistsByEmail(context.Context, user.Email) (bool, error) { return f.existsResp, nil }
func (f *fakeRepo) Delete(context.Context, user.ID) error                   { return nil }
func (f *fakeRepo) Restore(context.Context, user.ID) error                  { return nil }
func (f *fakeRepo) HardDelete(context.Context, user.ID) error               { return nil }
func (f *fakeRepo) CountActiveByRole(context.Context, user.Role) (int64, error) {
	return 1, nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(context.Context, string) (string, error) {
	return "$argon2id$v=19$m=65536,t=3,p=2$abc$def", nil
}
func (fakeHasher) Verify(context.Context, string, string) (bool, error) { return true, nil }
func (fakeHasher) NeedsRehash(string) bool                              { return false }

type fakePublisher struct{ published int }

func (f *fakePublisher) Publish(_ context.Context, e ...dshared.DomainEvent) error {
	f.published += len(e)
	return nil
}

// fakeTx, transaction'ı basitçe fn'i çalıştırarak taklit eder.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// fakeRoleChecker, yalnızca sistem rollerini (admin/user) geçerli kabul eder.
type fakeRoleChecker struct{}

func (fakeRoleChecker) RoleExists(_ context.Context, role string) (bool, error) {
	return role == "admin" || role == "user", nil
}

var (
	_ user.Repository          = (*fakeRepo)(nil)
	_ appshared.PasswordHasher = fakeHasher{}
	_ appshared.EventPublisher = (*fakePublisher)(nil)
	_ appshared.TxManager      = fakeTx{}
)

func TestRegisterHandler_Success(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{existsResp: false}
	pub := &fakePublisher{}
	h := appuser.NewRegisterHandler(repo, fakeHasher{}, pub, fakeTx{}, appuser.NewLocalePolicy("tr", []string{"tr", "en"}), fakeRoleChecker{})

	view, err := h.Handle(context.Background(), appuser.RegisterCommand{
		Email: "new@user.com", Name: "Yeni", Password: "password123", Role: "user",
	})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if view.Email != "new@user.com" {
		t.Errorf("email = %q", view.Email)
	}
	if repo.saved == nil {
		t.Error("kullanıcı kaydedilmeliydi")
	}
	if pub.published == 0 {
		t.Error("domain event yayınlanmalıydı")
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{existsResp: true} // e-posta zaten var
	h := appuser.NewRegisterHandler(repo, fakeHasher{}, &fakePublisher{}, fakeTx{}, appuser.NewLocalePolicy("tr", []string{"tr", "en"}), fakeRoleChecker{})

	_, err := h.Handle(context.Background(), appuser.RegisterCommand{
		Email: "dup@user.com", Name: "Ad", Password: "password123", Role: "user",
	})
	if !errors.Is(err, user.ErrEmailAlreadyExists) {
		t.Errorf("ErrEmailAlreadyExists bekleniyordu, alınan: %v", err)
	}
}

func TestRegisterHandler_InvalidRole(t *testing.T) {
	t.Parallel()
	h := appuser.NewRegisterHandler(&fakeRepo{}, fakeHasher{}, &fakePublisher{}, fakeTx{}, appuser.NewLocalePolicy("tr", []string{"tr", "en"}), fakeRoleChecker{})
	_, err := h.Handle(context.Background(), appuser.RegisterCommand{
		Email: "x@y.com", Name: "Ad", Password: "password123", Role: "superuser",
		AllowPrivilegedRole: true,
	})
	if err == nil {
		t.Error("geçersiz rol için hata bekleniyordu")
	}
}

func TestRegisterHandler_PublicIgnoresAdminRole(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{existsResp: false}
	h := appuser.NewRegisterHandler(repo, fakeHasher{}, &fakePublisher{}, fakeTx{}, appuser.NewLocalePolicy("tr", []string{"tr", "en"}), fakeRoleChecker{})
	_, err := h.Handle(context.Background(), appuser.RegisterCommand{
		Email: "x@y.com", Name: "Ad", Password: "password123", Role: "admin",
		AllowPrivilegedRole: false,
	})
	if err != nil {
		t.Fatalf("hata: %v", err)
	}
	if repo.saved.Role() != user.RoleUser {
		t.Errorf("public kayıt user rolü almalı, alınan: %s", repo.saved.Role())
	}
}
