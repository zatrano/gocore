package user

import "github.com/zatrano/gocore/internal/domain/shared"

// Event adları — event-driven entegrasyonlarda topic/routing key olarak da
// kullanılabilir.
const (
	EventRegistered      = "user.registered"
	EventActivated       = "user.activated"
	EventDeactivated     = "user.deactivated"
	EventEmailChanged    = "user.email_changed"
	EventPhoneChanged    = "user.phone_changed"
	EventNameChanged     = "user.name_changed"
	EventDeleted         = "user.deleted"
	EventRestored        = "user.restored"
	EventLocaleChanged   = "user.locale_changed"
	EventEmailVerified   = "user.email_verified"
	EventPasswordChanged = "user.password_changed"
	EventMFAEnabled      = "user.mfa_enabled"
	EventMFADisabled     = "user.mfa_disabled"
	EventRoleChanged     = "user.role_changed"
)

// RegisteredEvent, yeni bir kullanıcı kaydolduğunda üretilir.
type RegisteredEvent struct {
	shared.BaseEvent
	Email           string
	Name            string
	PreferredLocale string
}

// ActivatedEvent, kullanıcı aktifleştirildiğinde üretilir.
type ActivatedEvent struct {
	shared.BaseEvent
}

// DeactivatedEvent, kullanıcı pasifleştirildiğinde üretilir.
type DeactivatedEvent struct {
	shared.BaseEvent
}

// EmailChangedEvent, kullanıcı e-postasını değiştirdiğinde üretilir.
type EmailChangedEvent struct {
	shared.BaseEvent
	OldEmail string
	NewEmail string
}

// NameChangedEvent, kullanıcı adı değiştiğinde üretilir.
type NameChangedEvent struct {
	shared.BaseEvent
	OldName string
	NewName string
}

// PhoneChangedEvent, kullanıcı telefonunu değiştirdiğinde üretilir.
type PhoneChangedEvent struct {
	shared.BaseEvent
	OldPhone string
	NewPhone string
}

// DeletedEvent, kullanıcı yazılımsal olarak silindiğinde (soft delete) üretilir.
type DeletedEvent struct {
	shared.BaseEvent
}

// RestoredEvent, yazılımsal silme geri alındığında üretilir.
type RestoredEvent struct {
	shared.BaseEvent
}

// LocaleChangedEvent, kullanıcının dil tercihi değiştiğinde üretilir.
type LocaleChangedEvent struct {
	shared.BaseEvent
	OldLocale string
	NewLocale string
}

// EmailVerifiedEvent, kullanıcı e-posta adresini doğruladığında üretilir.
type EmailVerifiedEvent struct {
	shared.BaseEvent
}

// PasswordChangedEvent, kullanıcının şifresi değiştiğinde üretilir (değişiklik,
// sıfırlama). Güvenlik bildirimi ve oturum iptali tetikleyebilir.
type PasswordChangedEvent struct {
	shared.BaseEvent
}

// MFAEnabledEvent, iki adımlı doğrulama etkinleştirildiğinde üretilir.
type MFAEnabledEvent struct {
	shared.BaseEvent
}

// MFADisabledEvent, iki adımlı doğrulama kapatıldığında üretilir.
type MFADisabledEvent struct {
	shared.BaseEvent
}

// RoleChangedEvent, kullanıcının rolü değiştiğinde üretilir. Oturum iptali ve
// denetim kaydı tetikleyebilir.
type RoleChangedEvent struct {
	shared.BaseEvent
	OldRole string
	NewRole string
}

// NewRestoredEvent, aggregate yeniden yüklenmeden (ör. doğrudan repository
// restore akışında) RestoredEvent üretmek için yardımcıdır.
func NewRestoredEvent(userID string) RestoredEvent {
	return RestoredEvent{BaseEvent: shared.NewBaseEvent(EventRestored, userID)}
}

// AuditedEventNames, denetim kaydına yazılacak kullanıcı domain event adları.
func AuditedEventNames() []string {
	return []string{
		EventRegistered, EventActivated, EventDeactivated,
		EventEmailChanged, EventPhoneChanged, EventNameChanged,
		EventDeleted, EventRestored, EventLocaleChanged,
		EventEmailVerified, EventPasswordChanged,
		EventMFAEnabled, EventMFADisabled, EventRoleChanged,
	}
}
