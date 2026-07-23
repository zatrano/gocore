package user_test

import (
	"errors"
	"testing"

	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/domain/user"
)

func mustHashed(t *testing.T) user.HashedPassword {
	t.Helper()
	p, err := user.NewHashedPassword("$argon2id$v=19$m=65536,t=3,p=2$abc$def")
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	return p
}

func mustLocale(t *testing.T, raw string) user.PreferredLocale {
	t.Helper()
	loc, err := user.ParsePreferredLocale(raw)
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	return loc
}

func TestNewPhone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    string
	}{
		{"boş", "", false, ""},
		{"e164", "+905551112233", false, "+905551112233"},
		{"yerel 0", "05551112233", false, "+905551112233"},
		{"10 hane", "5551112233", false, "+905551112233"},
		{"boşluklu", "+90 555 111 22 33", false, "+905551112233"},
		{"geçersiz", "abc", true, ""},
		{"çok kısa", "+123", true, ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := user.NewPhone(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("hata bekleniyordu")
				}
				return
			}
			if err != nil {
				t.Fatalf("beklenmeyen hata: %v", err)
			}
			if got.String() != tt.want {
				t.Errorf("phone = %q, beklenen %q", got.String(), tt.want)
			}
		})
	}
}

func TestNewEmail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    string
	}{
		{"geçerli", "User@Example.COM", false, "user@example.com"},
		{"boşluklu normalize", "  a@b.com ", false, "a@b.com"},
		{"boş", "", true, ""},
		{"geçersiz format", "not-an-email", true, ""},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := user.NewEmail(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("hata bekleniyordu, alınmadı")
				}
				return
			}
			if err != nil {
				t.Fatalf("beklenmeyen hata: %v", err)
			}
			if got.String() != tt.want {
				t.Errorf("email = %q, beklenen %q", got.String(), tt.want)
			}
		})
	}
}

func TestRegister_InactiveByDefault_And_RecordsEvent(t *testing.T) {
	t.Parallel()
	email, _ := user.NewEmail("new@user.com")
	u, err := user.Register(email, "Yeni Kullanıcı", mustHashed(t), user.RoleUser, mustLocale(t, "tr"), user.Phone{})
	if err != nil {
		t.Fatalf("beklenmeyen hata: %v", err)
	}
	if u.IsActive() {
		t.Error("yeni kullanıcı pasif olmalı")
	}
	events := u.PullEvents()
	if len(events) != 1 || events[0].EventName() != user.EventRegistered {
		t.Errorf("RegisteredEvent bekleniyordu, alınan: %+v", events)
	}
}

func TestActivate_IdempotencyGuard(t *testing.T) {
	t.Parallel()
	email, _ := user.NewEmail("a@a.com")
	u, _ := user.Register(email, "Ad", mustHashed(t), user.RoleUser, mustLocale(t, "tr"), user.Phone{})
	_ = u.PullEvents()

	if err := u.Activate(); err != nil {
		t.Fatalf("ilk aktivasyon başarısız: %v", err)
	}
	if !u.IsActive() {
		t.Error("kullanıcı aktif olmalı")
	}
	// İkinci kez aktivasyon çakışma hatası vermeli.
	err := u.Activate()
	de, ok := shared.AsDomainError(err)
	if !ok || de.Kind != shared.KindConflict {
		t.Errorf("çakışma hatası bekleniyordu, alınan: %v", err)
	}
}

func TestDelete_SoftDeleteLifecycle(t *testing.T) {
	t.Parallel()
	email, _ := user.NewEmail("del@user.com")
	u, _ := user.Register(email, "Silinecek", mustHashed(t), user.RoleUser, mustLocale(t, "tr"), user.Phone{})
	_ = u.PullEvents()

	if u.IsDeleted() {
		t.Fatal("yeni kullanıcı silinmiş olmamalı")
	}

	if err := u.Delete(); err != nil {
		t.Fatalf("silme başarısız: %v", err)
	}
	if !u.IsDeleted() || u.DeletedAt() == nil {
		t.Error("kullanıcı silinmiş olarak işaretlenmeli")
	}
	events := u.PullEvents()
	if len(events) != 1 || events[0].EventName() != user.EventDeleted {
		t.Errorf("DeletedEvent bekleniyordu, alınan: %+v", events)
	}

	// İkinci silme çakışma hatası vermeli.
	if err := u.Delete(); !errors.Is(err, user.ErrAlreadyDeleted) {
		t.Errorf("ErrAlreadyDeleted bekleniyordu, alınan: %v", err)
	}
}

func TestRestore_Lifecycle(t *testing.T) {
	t.Parallel()
	email, _ := user.NewEmail("res@user.com")
	u, _ := user.Register(email, "Geri", mustHashed(t), user.RoleUser, mustLocale(t, "tr"), user.Phone{})
	_ = u.PullEvents()

	// Silinmemişken restore hata vermeli.
	if err := u.Restore(); !errors.Is(err, user.ErrNotDeleted) {
		t.Errorf("ErrNotDeleted bekleniyordu, alınan: %v", err)
	}

	_ = u.Delete()
	_ = u.PullEvents()

	if err := u.Restore(); err != nil {
		t.Fatalf("restore başarısız: %v", err)
	}
	if u.IsDeleted() {
		t.Error("restore sonrası kullanıcı canlı olmalı")
	}
	events := u.PullEvents()
	if len(events) != 1 || events[0].EventName() != user.EventRestored {
		t.Errorf("RestoredEvent bekleniyordu, alınan: %+v", events)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	t.Parallel()
	email, _ := user.NewEmail("a@a.com")
	_, err := user.Register(email, "   ", mustHashed(t), user.RoleUser, mustLocale(t, "tr"), user.Phone{})
	if !errors.Is(err, user.ErrNameRequired) {
		t.Errorf("ErrNameRequired bekleniyordu, alınan: %v", err)
	}
}
