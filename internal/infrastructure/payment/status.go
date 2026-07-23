package payment

import (
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	"github.com/zatrano/gocore/internal/infrastructure/config"
)

// IntegrationStatus, ödeme sağlayıcılarının yapılandırma durumunu taşır.
type IntegrationStatus struct {
	IyzicoConfigured bool
	MokaConfigured   bool
}

// IntegrationStatusFromConfig, ortam değişkenlerinden yapılandırma durumunu üretir.
func IntegrationStatusFromConfig(cfg config.Payment) IntegrationStatus {
	return IntegrationStatus{
		IyzicoConfigured: IyzicoConfigured(cfg),
		MokaConfigured:   MokaConfigured(cfg),
	}
}

// ToApplication, application katmanı DTO'suna dönüştürür.
func (s IntegrationStatus) ToApplication() appsettings.PaymentIntegrationStatus {
	return appsettings.PaymentIntegrationStatus{
		IyzicoConfigured: s.IyzicoConfigured,
		MokaConfigured:   s.MokaConfigured,
	}
}

// IyzicoConfigured, Iyzico kimlik bilgilerinin dolu olup olmadığını döner.
func IyzicoConfigured(cfg config.Payment) bool {
	return cfg.IyzicoAPIKey != "" && cfg.IyzicoSecretKey.Value() != ""
}

// MokaConfigured, Moka kimlik bilgilerinin dolu olup olmadığını döner.
func MokaConfigured(cfg config.Payment) bool {
	return cfg.MokaDealerCode != "" && cfg.MokaUsername != "" && cfg.MokaPassword.Value() != ""
}
