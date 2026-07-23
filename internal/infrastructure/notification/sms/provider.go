package sms

import (
	"context"
	"fmt"
	"log/slog"

	appsettings "github.com/zatrano/gocore/internal/application/settings"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// Provider, tek bir SMS sağlayıcısını temsil eden porttur.
type Provider interface {
	Name() string
	Send(ctx context.Context, to, body string) error
}

// ProviderError, sağlayıcı kaynaklı gönderim hatasını taşır.
type ProviderError struct {
	Provider string
	Code     string
	Message  string
}

func (e *ProviderError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("sms %s [%s]: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("sms %s: %s", e.Provider, e.Message)
}

// IntegrationStatus, SMS sağlayıcılarının yapılandırma durumunu taşır.
type IntegrationStatus struct {
	NetgsmConfigured       bool
	IletimerkeziConfigured bool
}

// IntegrationStatusFromConfig, ortam değişkenlerinden yapılandırma durumunu üretir.
func IntegrationStatusFromConfig(cfg config.Notify) IntegrationStatus {
	return IntegrationStatus{
		NetgsmConfigured:       NetgsmConfigured(cfg),
		IletimerkeziConfigured: IletimerkeziConfigured(cfg),
	}
}

// ToApplication, application katmanı DTO'suna dönüştürür.
func (s IntegrationStatus) ToApplication() appsettings.SMSIntegrationStatus {
	return appsettings.SMSIntegrationStatus{
		NetgsmConfigured:       s.NetgsmConfigured,
		IletimerkeziConfigured: s.IletimerkeziConfigured,
	}
}

// NewProviders, yapılandırılmış SMS sağlayıcılarını üretir (factory).
func NewProviders(cfg config.Notify, log *slog.Logger) []Provider {
	return []Provider{
		NewNetgsm(cfg, log),
		NewIletimerkezi(cfg, log),
	}
}

// BuildRegistry, tüm sağlayıcıları dashboard seçimine göre yönlendiren registry kurar.
func BuildRegistry(cfg config.Notify, settings *appsettings.Service, log *slog.Logger) *Registry {
	return NewRegistry(NewProviders(cfg, log), settings, log)
}

// NetgsmConfigured, Netgsm kimlik bilgilerinin dolu olup olmadığını döner.
func NetgsmConfigured(cfg config.Notify) bool {
	return cfg.NetgsmUser != "" && cfg.NetgsmPassword.Value() != "" && cfg.SMSFrom != ""
}

// IletimerkeziConfigured, İleti Merkezi kimlik bilgilerinin dolu olup olmadığını döner.
func IletimerkeziConfigured(cfg config.Notify) bool {
	return cfg.IletimerkeziKey != "" && cfg.IletimerkeziHash.Value() != "" && cfg.SMSFrom != ""
}
