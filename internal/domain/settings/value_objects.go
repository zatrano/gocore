package settings

import "slices"

// SettingKey, uygulama ayarı anahtarını temsil eder.
type SettingKey string

const (
	KeySMSActiveProvider     SettingKey = "sms.active_provider"
	KeyPaymentActiveProvider SettingKey = "payment.active_provider"
)

func (k SettingKey) String() string { return string(k) }

// SMSProvider, aktif SMS sağlayıcı adını temsil eder.
type SMSProvider string

const (
	ProviderNetgsm       SMSProvider = "netgsm"
	ProviderIletimerkezi SMSProvider = "iletimerkezi"
)

// AllSMSProviders, dashboard'da seçilebilir sağlayıcı listesidir (sıra önemli).
var AllSMSProviders = []SMSProvider{ProviderNetgsm, ProviderIletimerkezi}

// ParseSMSProvider, sağlayıcı adını doğrular.
func ParseSMSProvider(raw string) (SMSProvider, error) {
	p := SMSProvider(raw)
	if !IsValidSMSProvider(p) {
		return "", ErrInvalidSMSProvider
	}
	return p, nil
}

// IsValidSMSProvider, sağlayıcının desteklenip desteklenmediğini döner.
func IsValidSMSProvider(p SMSProvider) bool {
	return slices.Contains(AllSMSProviders, p)
}

func (p SMSProvider) String() string { return string(p) }

// PaymentProvider, aktif ödeme sağlayıcı adını temsil eder.
type PaymentProvider string

const (
	ProviderIyzico PaymentProvider = "iyzico"
	ProviderMoka   PaymentProvider = "moka"
)

// AllPaymentProviders, dashboard'da seçilebilir ödeme sağlayıcı listesidir (sıra önemli).
var AllPaymentProviders = []PaymentProvider{ProviderIyzico, ProviderMoka}

// ParsePaymentProvider, sağlayıcı adını doğrular.
func ParsePaymentProvider(raw string) (PaymentProvider, error) {
	p := PaymentProvider(raw)
	if !IsValidPaymentProvider(p) {
		return "", ErrInvalidPaymentProvider
	}
	return p, nil
}

// IsValidPaymentProvider, sağlayıcının desteklenip desteklenmediğini döner.
func IsValidPaymentProvider(p PaymentProvider) bool {
	return slices.Contains(AllPaymentProviders, p)
}

func (p PaymentProvider) String() string { return string(p) }
