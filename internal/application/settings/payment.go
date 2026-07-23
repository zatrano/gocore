package settings

import (
	"context"

	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
)

// PaymentIntegrationStatus, ödeme sağlayıcılarının ortam değişkeni yapılandırma durumudur.
type PaymentIntegrationStatus struct {
	IyzicoConfigured bool
	MokaConfigured   bool
}

// PaymentProviderRow, liste sayfası satırıdır.
type PaymentProviderRow struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Configured  bool   `json:"configured"`
}

// PaymentView, dashboard/API için ödeme ayar özetidir.
type PaymentView struct {
	ActiveProvider   string               `json:"active_provider"`
	Providers        []PaymentProviderRow `json:"providers"`
	IyzicoConfigured bool                 `json:"iyzico_configured"`
	MokaConfigured   bool                 `json:"moka_configured"`
}

func paymentProviderConfigured(p domainsettings.PaymentProvider, st PaymentIntegrationStatus) bool {
	switch p {
	case domainsettings.ProviderIyzico:
		return st.IyzicoConfigured
	case domainsettings.ProviderMoka:
		return st.MokaConfigured
	default:
		return false
	}
}

// GetPaymentView, mevcut ödeme ayar özetini döner.
func (s *Service) GetPaymentView(ctx context.Context, st PaymentIntegrationStatus) (PaymentView, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return PaymentView{}, err
	}
	active := settings.PaymentActiveProvider().String()
	rows := make([]PaymentProviderRow, len(domainsettings.AllPaymentProviders))
	for i, p := range domainsettings.AllPaymentProviders {
		rows[i] = PaymentProviderRow{
			Name:        p.String(),
			Label:       paymentProviderLabel(p),
			Description: paymentProviderDescription(p),
			Active:      p.String() == active,
			Configured:  paymentProviderConfigured(p, st),
		}
	}
	return PaymentView{
		ActiveProvider:   active,
		Providers:        rows,
		IyzicoConfigured: st.IyzicoConfigured,
		MokaConfigured:   st.MokaConfigured,
	}, nil
}

// GetPaymentProvider, tek ödeme sağlayıcı satırını döner.
func (s *Service) GetPaymentProvider(ctx context.Context, name string, st PaymentIntegrationStatus) (PaymentProviderRow, error) {
	p, err := domainsettings.ParsePaymentProvider(name)
	if err != nil {
		return PaymentProviderRow{}, err
	}
	view, err := s.GetPaymentView(ctx, st)
	if err != nil {
		return PaymentProviderRow{}, err
	}
	for _, row := range view.Providers {
		if row.Name == p.String() {
			return row, nil
		}
	}
	return PaymentProviderRow{}, domainsettings.ErrInvalidPaymentProvider
}

func paymentProviderLabel(p domainsettings.PaymentProvider) string {
	switch p {
	case domainsettings.ProviderIyzico:
		return "Iyzico"
	case domainsettings.ProviderMoka:
		return "Moka"
	default:
		return p.String()
	}
}

func paymentProviderDescription(p domainsettings.PaymentProvider) string {
	switch p {
	case domainsettings.ProviderIyzico:
		return "Iyzico ödeme altyapısı"
	case domainsettings.ProviderMoka:
		return "Moka United ödeme altyapısı"
	default:
		return ""
	}
}
