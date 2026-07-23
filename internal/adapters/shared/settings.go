package shared

import (
	"errors"

	appsettings "github.com/zatrano/gocore/internal/application/settings"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	smssvc "github.com/zatrano/gocore/internal/infrastructure/notification/sms"
	paymentsvc "github.com/zatrano/gocore/internal/infrastructure/payment"
)

var (
	errNetgsmNotConfigured       = errors.New("netgsm seçmek için NOTIFY_NETGSM_* ve NOTIFY_SMS_FROM ortam değişkenleri tanımlı olmalıdır")
	errIletimerkeziNotConfigured = errors.New("ileti merkezi seçmek için NOTIFY_ILETIMERKEZI_* ve NOTIFY_SMS_FROM ortam değişkenleri tanımlı olmalıdır")
)

// SMSIntegrationStatus, ortam değişkenlerinden SMS entegrasyon durumunu döner.
func SMSIntegrationStatus(cfg config.Notify) appsettings.SMSIntegrationStatus {
	return smssvc.IntegrationStatusFromConfig(cfg).ToApplication()
}

// ValidateSMSActivation, aktif SMS sağlayıcısı seçiminin yapılandırma koşullarını doğrular.
func ValidateSMSActivation(provider string, st appsettings.SMSIntegrationStatus) error {
	switch provider {
	case domainsettings.ProviderNetgsm.String():
		if !st.NetgsmConfigured {
			return errNetgsmNotConfigured
		}
	case domainsettings.ProviderIletimerkezi.String():
		if !st.IletimerkeziConfigured {
			return errIletimerkeziNotConfigured
		}
	}
	return nil
}

// SMSProviderPayload, API/web ortak SMS sağlayıcı detay yanıtıdır.
type SMSProviderPayload struct {
	Provider               appsettings.SMSProviderRow `json:"provider"`
	NetgsmConfigured       bool                       `json:"netgsm_configured"`
	IletimerkeziConfigured bool                       `json:"iletimerkezi_configured"`
}

// NewSMSProviderPayload, sağlayıcı satırını yapılandırma durumuyla birlikte sarar.
func NewSMSProviderPayload(row appsettings.SMSProviderRow, st appsettings.SMSIntegrationStatus) SMSProviderPayload {
	return SMSProviderPayload{
		Provider:               row,
		NetgsmConfigured:       st.NetgsmConfigured,
		IletimerkeziConfigured: st.IletimerkeziConfigured,
	}
}

// SMSListPayload, API/web ortak SMS liste yanıtıdır.
type SMSListPayload struct {
	ActiveProvider         string                       `json:"active_provider"`
	Providers              []appsettings.SMSProviderRow `json:"providers"`
	NetgsmConfigured       bool                         `json:"netgsm_configured"`
	IletimerkeziConfigured bool                         `json:"iletimerkezi_configured"`
}

// NewSMSListPayload, liste satırlarını yapılandırma durumuyla birlikte sarar.
func NewSMSListPayload(rows []appsettings.SMSProviderRow, active string, st appsettings.SMSIntegrationStatus) SMSListPayload {
	return SMSListPayload{
		ActiveProvider:         active,
		Providers:              rows,
		NetgsmConfigured:       st.NetgsmConfigured,
		IletimerkeziConfigured: st.IletimerkeziConfigured,
	}
}

// PaymentProviderPayload, API/web ortak ödeme sağlayıcı detay yanıtıdır.
type PaymentProviderPayload struct {
	Provider appsettings.PaymentProviderRow `json:"provider"`
}

// NewPaymentProviderPayload, ödeme sağlayıcı satırını sarar.
func NewPaymentProviderPayload(row appsettings.PaymentProviderRow) PaymentProviderPayload {
	return PaymentProviderPayload{Provider: row}
}

var (
	errIyzicoNotConfigured = errors.New("iyzico seçmek için PAYMENT_IYZICO_API_KEY ve PAYMENT_IYZICO_SECRET_KEY ortam değişkenleri tanımlı olmalıdır")
	errMokaNotConfigured   = errors.New("moka seçmek için PAYMENT_MOKA_DEALER_CODE, PAYMENT_MOKA_USERNAME ve PAYMENT_MOKA_PASSWORD ortam değişkenleri tanımlı olmalıdır")
)

// PaymentIntegrationStatus, ortam değişkenlerinden ödeme entegrasyon durumunu döner.
func PaymentIntegrationStatus(cfg config.Payment) appsettings.PaymentIntegrationStatus {
	return paymentsvc.IntegrationStatusFromConfig(cfg).ToApplication()
}

// ValidatePaymentActivation, aktif ödeme sağlayıcısı seçiminin yapılandırma koşullarını doğrular.
func ValidatePaymentActivation(provider string, st appsettings.PaymentIntegrationStatus) error {
	switch provider {
	case domainsettings.ProviderIyzico.String():
		if !st.IyzicoConfigured {
			return errIyzicoNotConfigured
		}
	case domainsettings.ProviderMoka.String():
		if !st.MokaConfigured {
			return errMokaNotConfigured
		}
	}
	return nil
}

// PaymentListPayload, API ortak ödeme liste yanıtıdır (PaymentView ile aynı).
type PaymentListPayload = appsettings.PaymentView
