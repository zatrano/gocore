package auth

import "github.com/zatrano/gocore/internal/domain/shared"

const ( //nolint:gosec // G101 — domain event sabitleri, kimlik bilgisi değil.
	EventOneTimeTokenCreated = "auth.one_time_token_created"
	EventOneTimeTokenUsed    = "auth.one_time_token_used"
	EventLoginSucceeded      = "auth.login_succeeded"
	EventLoginFailed         = "auth.login_failed"
	EventLogout              = "auth.logout"
)

// OneTimeTokenCreatedEvent, yeni tek kullanımlık token üretildiğinde kaydedilir.
type OneTimeTokenCreatedEvent struct {
	shared.BaseEvent
	UserID  string
	Purpose string
}

// OneTimeTokenUsedEvent, token tüketildiğinde kaydedilir.
type OneTimeTokenUsedEvent struct {
	shared.BaseEvent
	UserID  string
	Purpose string
}

// LoginSucceededEvent, başarılı kimlik doğrulamada üretilir.
type LoginSucceededEvent struct {
	shared.BaseEvent
	Email    string
	Provider string // password | google | github | ...
}

// NewLoginSucceededEvent, başarılı giriş olayı üretir.
func NewLoginSucceededEvent(userID, email, provider string) LoginSucceededEvent {
	return LoginSucceededEvent{
		BaseEvent: shared.NewBaseEvent(EventLoginSucceeded, userID),
		Email:     email,
		Provider:  provider,
	}
}

// LoginFailedEvent, başarısız giriş denemesinde üretilir.
type LoginFailedEvent struct {
	shared.BaseEvent
	Email  string
	Reason string
}

// NewLoginFailedEvent, başarısız giriş olayı üretir.
// Kullanıcı bilinmiyorsa aggregateID boş kalabilir.
func NewLoginFailedEvent(userID, email, reason string) LoginFailedEvent {
	return LoginFailedEvent{
		BaseEvent: shared.NewBaseEvent(EventLoginFailed, userID),
		Email:     email,
		Reason:    reason,
	}
}

// LogoutEvent, oturum kapatıldığında üretilir.
type LogoutEvent struct {
	shared.BaseEvent
	Email string
}

// NewLogoutEvent, çıkış olayı üretir.
func NewLogoutEvent(userID, email string) LogoutEvent {
	return LogoutEvent{
		BaseEvent: shared.NewBaseEvent(EventLogout, userID),
		Email:     email,
	}
}

// AuditedEventNames, denetim kapsamındaki auth olayları.
func AuditedEventNames() []string {
	return []string{
		EventOneTimeTokenCreated, EventOneTimeTokenUsed,
		EventLoginSucceeded, EventLoginFailed, EventLogout,
	}
}
