package user

import (
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// User, kullanıcı bounded context'inin aggregate root'udur. Tüm invariant'lar
// (iş kuralları) bu tip üzerinden korunur; alanlar dışarıya kapalıdır ve yalnızca
// davranış metodları aracılığıyla değiştirilir. Bu, geçersiz durumları en baştan
// imkânsız kılar.
type User struct {
	shared.EventRecorder

	id              ID
	email           Email
	phone           Phone
	name            string
	password        HashedPassword
	role            Role
	active          bool
	emailVerified   bool
	mfaEnabled      bool
	mfaSecret       string // TOTP paylaşılan sırrı (base32); boş = yapılandırılmamış
	preferredLocale PreferredLocale
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time // nil = canlı, dolu = yazılımsal silinmiş (soft delete)
}

// Register, yeni bir kullanıcı aggregate'i oluşturur ve RegisteredEvent üretir.
// Bu, kullanıcının doğduğu tek yer olan factory metodudur. phone opsiyoneldir.
func Register(email Email, name string, password HashedPassword, role Role, locale PreferredLocale, phone Phone) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNameRequired
	}
	if password.IsZero() {
		return nil, ErrInvalidPasswordHash
	}

	now := time.Now().UTC()
	u := &User{
		id:              NewID(),
		email:           email,
		phone:           phone,
		name:            name,
		password:        password,
		role:            role,
		active:          false,
		preferredLocale: locale,
		createdAt:       now,
		updatedAt:       now,
	}
	u.Record(RegisteredEvent{
		BaseEvent:       shared.NewBaseEvent(EventRegistered, u.id.String()),
		Email:           email.String(),
		Name:            name,
		PreferredLocale: locale.String(),
	})
	return u, nil
}

// Hydrate, persistence katmanından okunan verilerle bir User'ı yeniden
// oluşturur. Yalnızca repository tarafından kullanılmalıdır; hiçbir event
// üretmez ve iş kuralı çalıştırmaz (veri zaten geçerli kabul edilir).
func Hydrate(
	id ID, email Email, phone Phone, name string, password HashedPassword,
	role Role, active, emailVerified, mfaEnabled bool, mfaSecret string,
	locale PreferredLocale,
	createdAt, updatedAt time.Time, deletedAt *time.Time,
) *User {
	return &User{
		id:              id,
		email:           email,
		phone:           phone,
		name:            name,
		password:        password,
		role:            role,
		active:          active,
		emailVerified:   emailVerified,
		mfaEnabled:      mfaEnabled,
		mfaSecret:       mfaSecret,
		preferredLocale: locale,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		deletedAt:       deletedAt,
	}
}

// --- Davranışlar (invariant koruyan mutasyonlar) ---

// Activate, hesabı aktifleştirir. Zaten aktifse hata döner.
func (u *User) Activate() error {
	if u.active {
		return ErrAlreadyActive
	}
	u.active = true
	u.touch()
	u.Record(ActivatedEvent{BaseEvent: shared.NewBaseEvent(EventActivated, u.id.String())})
	return nil
}

// Deactivate, hesabı pasifleştirir.
func (u *User) Deactivate() error {
	if !u.active {
		return ErrInactive
	}
	u.active = false
	u.touch()
	u.Record(DeactivatedEvent{BaseEvent: shared.NewBaseEvent(EventDeactivated, u.id.String())})
	return nil
}

// ChangeEmail, e-postayı değiştirir ve EmailChangedEvent üretir.
func (u *User) ChangeEmail(newEmail Email) {
	if u.email.String() == newEmail.String() {
		return
	}
	old := u.email
	u.email = newEmail
	u.touch()
	u.Record(EmailChangedEvent{
		BaseEvent: shared.NewBaseEvent(EventEmailChanged, u.id.String()),
		OldEmail:  old.String(),
		NewEmail:  newEmail.String(),
	})
}

// ChangePhone, telefon numarasını değiştirir (boş = kaldırma).
func (u *User) ChangePhone(newPhone Phone) {
	if u.phone.String() == newPhone.String() {
		return
	}
	old := u.phone
	u.phone = newPhone
	u.touch()
	u.Record(PhoneChangedEvent{
		BaseEvent: shared.NewBaseEvent(EventPhoneChanged, u.id.String()),
		OldPhone:  old.String(),
		NewPhone:  newPhone.String(),
	})
}

// ChangePreferredLocale, kullanıcının kalıcı dil tercihini günceller.
func (u *User) ChangePreferredLocale(loc PreferredLocale) {
	if u.preferredLocale.String() == loc.String() {
		return
	}
	old := u.preferredLocale.String()
	u.preferredLocale = loc
	u.touch()
	u.Record(LocaleChangedEvent{
		BaseEvent: shared.NewBaseEvent(EventLocaleChanged, u.id.String()),
		OldLocale: old,
		NewLocale: loc.String(),
	})
}

// Rename, kullanıcının adını değiştirir.
func (u *User) Rename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrNameRequired
	}
	if u.name == name {
		return nil
	}
	old := u.name
	u.name = name
	u.touch()
	u.Record(NameChangedEvent{
		BaseEvent: shared.NewBaseEvent(EventNameChanged, u.id.String()),
		OldName:   old,
		NewName:   name,
	})
	return nil
}

// SetPassword, yeni (hash'lenmiş) şifreyi atar ve PasswordChangedEvent üretir.
func (u *User) SetPassword(p HashedPassword) error {
	if p.IsZero() {
		return ErrInvalidPasswordHash
	}
	u.password = p
	u.touch()
	u.Record(PasswordChangedEvent{BaseEvent: shared.NewBaseEvent(EventPasswordChanged, u.id.String())})
	return nil
}

// RehashPassword, şifre hash'ini sessizce günceller (parametre güçlendirme /
// transparent rehash). SetPassword'ün aksine event üretmez: kullanıcı için
// görünür bir değişiklik değildir, yalnızca aynı şifrenin daha güçlü hash'idir.
func (u *User) RehashPassword(p HashedPassword) error {
	if p.IsZero() {
		return ErrInvalidPasswordHash
	}
	u.password = p
	u.touch()
	return nil
}

// VerifyEmail, e-posta adresini doğrulanmış olarak işaretler. Zaten
// doğrulanmışsa hata döner.
func (u *User) VerifyEmail() error {
	if u.emailVerified {
		return ErrEmailAlreadyVerified
	}
	u.emailVerified = true
	u.touch()
	u.Record(EmailVerifiedEvent{BaseEvent: shared.NewBaseEvent(EventEmailVerified, u.id.String())})
	return nil
}

// ConfigureMFA, TOTP sırrını atar (kurulum adımı). MFA henüz etkinleşmez;
// kullanıcı bir doğrulama koduyla EnableMFA çağrılana kadar pasiftir.
func (u *User) ConfigureMFA(secret string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ErrMFANotConfigured
	}
	u.mfaSecret = secret
	u.touch()
	return nil
}

// EnableMFA, yapılandırılmış TOTP sırrıyla iki adımlı doğrulamayı etkinleştirir.
func (u *User) EnableMFA() error {
	if u.mfaSecret == "" {
		return ErrMFANotConfigured
	}
	if u.mfaEnabled {
		return ErrMFAAlreadyEnabled
	}
	u.mfaEnabled = true
	u.touch()
	u.Record(MFAEnabledEvent{BaseEvent: shared.NewBaseEvent(EventMFAEnabled, u.id.String())})
	return nil
}

// DisableMFA, iki adımlı doğrulamayı kapatır ve sırrı siler.
func (u *User) DisableMFA() error {
	if !u.mfaEnabled {
		return ErrMFANotEnabled
	}
	u.mfaEnabled = false
	u.mfaSecret = ""
	u.touch()
	u.Record(MFADisabledEvent{BaseEvent: shared.NewBaseEvent(EventMFADisabled, u.id.String())})
	return nil
}

// ChangeRole, kullanıcının rolünü değiştirir ve RoleChangedEvent üretir.
// Aynı rol atanmışsa ErrSameRole döner.
func (u *User) ChangeRole(newRole Role) error {
	if u.role == newRole {
		return ErrSameRole
	}
	old := u.role
	u.role = newRole
	u.touch()
	u.Record(RoleChangedEvent{
		BaseEvent: shared.NewBaseEvent(EventRoleChanged, u.id.String()),
		OldRole:   old.String(),
		NewRole:   newRole.String(),
	})
	return nil
}

// Delete, kullanıcıyı yazılımsal olarak siler (soft delete). Kayıt fiziksel
// olarak durur ancak canlı sorgularda görünmez. Zaten silinmişse hata döner.
func (u *User) Delete() error {
	if u.deletedAt != nil {
		return ErrAlreadyDeleted
	}
	now := time.Now().UTC()
	u.deletedAt = &now
	u.touch()
	u.Record(DeletedEvent{BaseEvent: shared.NewBaseEvent(EventDeleted, u.id.String())})
	return nil
}

// Restore, yazılımsal silmeyi geri alır. Silinmemiş kayıtta hata döner.
func (u *User) Restore() error {
	if u.deletedAt == nil {
		return ErrNotDeleted
	}
	u.deletedAt = nil
	u.touch()
	u.Record(RestoredEvent{BaseEvent: shared.NewBaseEvent(EventRestored, u.id.String())})
	return nil
}

func (u *User) touch() { u.updatedAt = time.Now().UTC() }

// --- Getter'lar (salt okunur erişim) ---

func (u *User) ID() ID                           { return u.id }
func (u *User) Email() Email                     { return u.email }
func (u *User) Phone() Phone                     { return u.phone }
func (u *User) Name() string                     { return u.name }
func (u *User) Password() HashedPassword         { return u.password }
func (u *User) Role() Role                       { return u.role }
func (u *User) IsActive() bool                   { return u.active }
func (u *User) IsEmailVerified() bool            { return u.emailVerified }
func (u *User) MFAEnabled() bool                 { return u.mfaEnabled }
func (u *User) MFASecret() string                { return u.mfaSecret }
func (u *User) PreferredLocale() PreferredLocale { return u.preferredLocale }
func (u *User) CreatedAt() time.Time             { return u.createdAt }
func (u *User) UpdatedAt() time.Time             { return u.updatedAt }

// IsDeleted, kullanıcının yazılımsal olarak silinip silinmediğini döner.
func (u *User) IsDeleted() bool { return u.deletedAt != nil }

// DeletedAt, silinme zamanını döner (silinmemişse nil).
func (u *User) DeletedAt() *time.Time { return u.deletedAt }
