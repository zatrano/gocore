package settings

import (
	"context"

	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
)

// SMSIntegrationStatus, SMS sağlayıcılarının ortam değişkeni yapılandırma durumudur.
type SMSIntegrationStatus struct {
	NetgsmConfigured       bool
	IletimerkeziConfigured bool
}

// View, dashboard/API için SMS ayar özetidir.
type View struct {
	ActiveProvider         string   `json:"active_provider"`
	Providers              []string `json:"providers"`
	NetgsmConfigured       bool     `json:"netgsm_configured"`
	IletimerkeziConfigured bool     `json:"iletimerkezi_configured"`
}

// SMSProviderRow, liste sayfası satırıdır.
type SMSProviderRow struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Configured  bool   `json:"configured"`
}

func providerConfigured(p domainsettings.SMSProvider, st SMSIntegrationStatus) bool {
	switch p {
	case domainsettings.ProviderNetgsm:
		return st.NetgsmConfigured
	case domainsettings.ProviderIletimerkezi:
		return st.IletimerkeziConfigured
	default:
		return false
	}
}

// ListSMSProviders, SMS sağlayıcı listesini döner.
func (s *Service) ListSMSProviders(ctx context.Context, st SMSIntegrationStatus) ([]SMSProviderRow, string, error) {
	view, err := s.GetSMSView(ctx, st)
	if err != nil {
		return nil, "", err
	}
	rows := make([]SMSProviderRow, len(domainsettings.AllSMSProviders))
	for i, p := range domainsettings.AllSMSProviders {
		rows[i] = SMSProviderRow{
			Name:        p.String(),
			Label:       smsProviderLabel(p),
			Description: smsProviderDescription(p),
			Active:      p.String() == view.ActiveProvider,
			Configured:  providerConfigured(p, st),
		}
	}
	return rows, view.ActiveProvider, nil
}

// GetSMSProvider, tek SMS sağlayıcı satırını döner.
func (s *Service) GetSMSProvider(ctx context.Context, name string, st SMSIntegrationStatus) (SMSProviderRow, error) {
	p, err := domainsettings.ParseSMSProvider(name)
	if err != nil {
		return SMSProviderRow{}, err
	}
	rows, _, err := s.ListSMSProviders(ctx, st)
	if err != nil {
		return SMSProviderRow{}, err
	}
	for _, row := range rows {
		if row.Name == p.String() {
			return row, nil
		}
	}
	return SMSProviderRow{}, domainsettings.ErrInvalidSMSProvider
}

// GetSMSView, mevcut SMS ayar özetini döner.
func (s *Service) GetSMSView(ctx context.Context, st SMSIntegrationStatus) (View, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return View{}, err
	}
	providers := make([]string, len(domainsettings.AllSMSProviders))
	for i, p := range domainsettings.AllSMSProviders {
		providers[i] = p.String()
	}
	return View{
		ActiveProvider:         settings.SMSActiveProvider().String(),
		Providers:              providers,
		NetgsmConfigured:       st.NetgsmConfigured,
		IletimerkeziConfigured: st.IletimerkeziConfigured,
	}, nil
}

func smsProviderLabel(p domainsettings.SMSProvider) string {
	switch p {
	case domainsettings.ProviderNetgsm:
		return "Netgsm"
	case domainsettings.ProviderIletimerkezi:
		return "İleti Merkezi"
	default:
		return p.String()
	}
}

func smsProviderDescription(p domainsettings.SMSProvider) string {
	switch p {
	case domainsettings.ProviderNetgsm:
		return "Netgsm REST v2 entegrasyonu"
	case domainsettings.ProviderIletimerkezi:
		return "İleti Merkezi GET API entegrasyonu"
	default:
		return ""
	}
}
