package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/pagination"
)

type stubUserRepo struct {
	byID    map[string]*user.User
	byEmail map[string]*user.User
}

func (s *stubUserRepo) Save(context.Context, *user.User) error   { return nil }
func (s *stubUserRepo) Update(context.Context, *user.User) error { return nil }
func (s *stubUserRepo) FindByID(_ context.Context, id user.ID) (*user.User, error) {
	if u, ok := s.byID[id.String()]; ok {
		return u, nil
	}
	return nil, user.ErrNotFound
}
func (s *stubUserRepo) FindByIDIncludeDeleted(context.Context, user.ID) (*user.User, error) {
	return nil, user.ErrNotFound
}
func (s *stubUserRepo) FindByEmail(_ context.Context, email user.Email) (*user.User, error) {
	if u, ok := s.byEmail[email.String()]; ok {
		return u, nil
	}
	return nil, user.ErrNotFound
}
func (s *stubUserRepo) FindByIDs(context.Context, []user.ID) ([]*user.User, error) { return nil, nil }
func (s *stubUserRepo) List(context.Context, user.ListFilter, pagination.Request) (pagination.Page[*user.User], error) {
	return pagination.Page[*user.User]{}, nil
}
func (s *stubUserRepo) ExistsByEmail(context.Context, user.Email) (bool, error) { return false, nil }
func (s *stubUserRepo) Delete(context.Context, user.ID) error                   { return nil }
func (s *stubUserRepo) Restore(context.Context, user.ID) error                  { return nil }
func (s *stubUserRepo) HardDelete(context.Context, user.ID) error               { return nil }
func (s *stubUserRepo) CountActiveByRole(context.Context, user.Role) (int64, error) {
	return 0, nil
}

func activeTestUser(id user.ID, email string) *user.User {
	return activeTestUserWithLocale(id, email, "tr")
}

func activeTestUserWithLocale(id user.ID, email, locale string) *user.User {
	emailVO, _ := user.NewEmail(email)
	hashed, _ := user.NewHashedPassword("$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$hash")
	loc, _ := user.ParsePreferredLocale(locale)
	now := time.Now().UTC()
	return user.Hydrate(id, emailVO, user.Phone{}, "Test", hashed, user.RoleUser, true, true, false, "", loc, now, now, nil)
}

func TestUserRepoResolver_ResolveByUserID(t *testing.T) {
	id := user.NewID()
	u := activeTestUser(id, "alice@example.com")
	repo := &stubUserRepo{byID: map[string]*user.User{id.String(): u}}
	resolver := UserRepoResolver{Users: repo}

	got, err := resolver.Resolve(context.Background(), ChannelInApp, Recipient{UserID: id.String()})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.UserID != id.String() {
		t.Fatalf("UserID = %q, want %q", got.UserID, id.String())
	}
}

func TestUserRepoResolver_ResolveByEmail(t *testing.T) {
	id := user.NewID()
	u := activeTestUser(id, "alice@example.com")
	repo := &stubUserRepo{byEmail: map[string]*user.User{u.Email().String(): u}}
	resolver := UserRepoResolver{Users: repo}

	got, err := resolver.Resolve(context.Background(), ChannelInApp, Recipient{Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.UserID != id.String() {
		t.Fatalf("UserID = %q, want %q", got.UserID, id.String())
	}
}

func TestUserRepoResolver_NotFound(t *testing.T) {
	resolver := UserRepoResolver{Users: &stubUserRepo{}}
	_, err := resolver.Resolve(context.Background(), ChannelInApp, Recipient{Email: "missing@example.com"})
	if !errors.Is(err, ErrRecipientNotFound) {
		t.Fatalf("ErrRecipientNotFound bekleniyordu, alınan: %v", err)
	}
}

func TestUserRepoResolver_Required(t *testing.T) {
	resolver := UserRepoResolver{Users: &stubUserRepo{}}
	_, err := resolver.Resolve(context.Background(), ChannelInApp, Recipient{})
	if !errors.Is(err, ErrRecipientRequired) {
		t.Fatalf("ErrRecipientRequired bekleniyordu, alınan: %v", err)
	}
}

func TestUserRepoResolver_PassthroughOtherChannels(t *testing.T) {
	resolver := UserRepoResolver{Users: &stubUserRepo{}}
	in := Recipient{Email: "a@b.com", Phone: "+905551112233"}
	got, err := resolver.Resolve(context.Background(), ChannelEmail, in)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != in {
		t.Fatalf("bilinmeyen e-posta değiştirilmemeli: %+v", got)
	}
	gotSMS, err := resolver.Resolve(context.Background(), ChannelSMS, in)
	if err != nil {
		t.Fatalf("Resolve SMS: %v", err)
	}
	if gotSMS != in {
		t.Fatalf("sms kanalı değiştirilmemeli: %+v", gotSMS)
	}
}

func TestUserRepoResolver_EmailSetsPreferredLocale(t *testing.T) {
	id := user.NewID()
	u := activeTestUserWithLocale(id, "alice@example.com", "en")
	repo := &stubUserRepo{byEmail: map[string]*user.User{u.Email().String(): u}}
	resolver := UserRepoResolver{Users: repo}

	got, err := resolver.Resolve(context.Background(), ChannelEmail, Recipient{Email: "alice@example.com", Locale: "tr"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Locale != "en" {
		t.Fatalf("Locale = %q, want en (kullanıcı tercihi)", got.Locale)
	}
}
