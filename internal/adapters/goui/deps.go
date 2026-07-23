// Package goui implements the browser UI with GoUI components.
package goui

import (
	"time"

	"github.com/go-playground/validator/v10"

	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appauth "github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/application/authz"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	"github.com/zatrano/gocore/internal/infrastructure/security/turnstile"
	pkgi18n "github.com/zatrano/gocore/pkg/i18n"
	"github.com/zatrano/gocore/pkg/rbac"
)

// AuthDeps, kimlik doğrulama use-case'lerini gruplar.
type AuthDeps struct {
	Auth *appauth.Service
}

// UserDeps, kullanıcı yönetimi application servisini gruplar.
type UserDeps struct {
	Users *appuser.Service
}

// NotificationDeps, bildirim application servisini gruplar.
type NotificationDeps struct {
	Notifications *appnotif.Service
	// InboxRealtime, okundu işaretlemede genel /api/v1/ws hub'ına push eder.
	InboxRealtime interface{ NotifyInbox(userID string) }
}

// ContactDeps, iletişim formu use-case'lerini gruplar.
type ContactDeps struct {
	Contacts *appcontact.Service
}

// PaymentSettingsDeps, ayarlar ve ödeme use-case'lerini gruplar.
type PaymentSettingsDeps struct {
	Settings   *appsettings.Service
	Notify     config.Notify
	Payment    config.Payment
	ThreeDSSvc *apppayment.ThreeDSService
}

// AuditDeps, denetim use-case'lerini gruplar.
type AuditDeps struct {
	Audit *appaudit.Service
}

// Deps, GoUI bileşen fabrikalarının kullandığı application servisleridir.
// Alanlar gömülü gruplar üzerinden promote edilir (p.Deps.Auth vb. çalışır).
// Domain ve application paketleri UI transport'undan habersiz kalır.
type Deps struct {
	AuthDeps
	UserDeps
	NotificationDeps
	ContactDeps
	PaymentSettingsDeps
	AuditDeps

	Authz            *authz.Service
	Checker          rbac.Checker
	Storage          appshared.Storage
	Upload           *appupload.Service
	Validate         *validator.Validate
	Secure           bool
	AccessTTL        time.Duration
	MaxUpload        int64
	AllowedMIMEs     []string
	Locales          []string
	Turnstile        turnstile.Verifier
	TurnstileSiteKey string
	Cache            appshared.Cache
	Translator       *pkgi18n.Translator
	// RateLimit, GoUI WS mutasyonları için IP kotası (Fiber API limiter'ın karşılığı).
	RateLimit func(key string) bool
	// ViewsRoot, Blade tarzı .goui.html şablonlarının kök dizini (opsiyonel).
	// Boşsa kaynak ağacı veya gömülü views kullanılır.
	ViewsRoot string
}
