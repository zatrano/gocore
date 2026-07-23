package settings

import (
	"context"
	"sync"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
)

// Service, platform ayarları use-case'lerini yönetir (tek aggregate, paylaşımlı önbellek).
type Service struct {
	repo      domainsettings.Repository
	publisher appshared.EventPublisher
	tx        appshared.TxManager
	mu        sync.RWMutex
	cache     *domainsettings.PlatformSettings
}

// SettingsDeps, Service bağımlılıklarını gruplar.
type SettingsDeps struct {
	Repo      domainsettings.Repository
	Publisher appshared.EventPublisher
	Tx        appshared.TxManager
}

// NewService, servisi kurar.
func NewService(d SettingsDeps) *Service {
	return &Service{repo: d.Repo, publisher: d.Publisher, tx: d.Tx}
}

// ActiveProvider, aktif SMS sağlayıcı adını döner (SMS registry uyumluluğu).
func (s *Service) ActiveProvider(ctx context.Context) (string, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return domainsettings.ProviderNetgsm.String(), err
	}
	return settings.SMSActiveProvider().String(), nil
}

// SetSMSActiveProvider, aktif SMS sağlayıcısını günceller.
func (s *Service) SetSMSActiveProvider(ctx context.Context, provider string) error {
	p, err := domainsettings.ParseSMSProvider(provider)
	if err != nil {
		return err
	}
	apply := func(ctx context.Context) error {
		settings, err := s.loadFresh(ctx)
		if err != nil {
			return err
		}
		if err := settings.SetSMSActiveProvider(p); err != nil {
			return err
		}
		if err := s.repo.Set(ctx, domainsettings.KeySMSActiveProvider, p.String()); err != nil {
			return err
		}
		if s.publisher != nil {
			if err := s.publisher.Publish(ctx, settings.PullEvents()...); err != nil {
				return err
			}
		}
		s.store(settings)
		return nil
	}
	if s.tx != nil {
		return s.tx.WithinTx(ctx, apply)
	}
	return apply(ctx)
}

// SetPaymentActiveProvider, aktif ödeme sağlayıcısını günceller.
func (s *Service) SetPaymentActiveProvider(ctx context.Context, provider string) error {
	p, err := domainsettings.ParsePaymentProvider(provider)
	if err != nil {
		return err
	}
	apply := func(ctx context.Context) error {
		settings, err := s.loadFresh(ctx)
		if err != nil {
			return err
		}
		if err := settings.SetPaymentActiveProvider(p); err != nil {
			return err
		}
		if err := s.repo.Set(ctx, domainsettings.KeyPaymentActiveProvider, p.String()); err != nil {
			return err
		}
		if s.publisher != nil {
			if err := s.publisher.Publish(ctx, settings.PullEvents()...); err != nil {
				return err
			}
		}
		s.store(settings)
		return nil
	}
	if s.tx != nil {
		return s.tx.WithinTx(ctx, apply)
	}
	return apply(ctx)
}

func (s *Service) load(ctx context.Context) (*domainsettings.PlatformSettings, error) {
	s.mu.RLock()
	if s.cache != nil {
		cached := s.cache
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()
	return s.loadFresh(ctx)
}

func (s *Service) loadFresh(ctx context.Context) (*domainsettings.PlatformSettings, error) {
	smsRaw, err := s.repo.Get(ctx, domainsettings.KeySMSActiveProvider)
	if err != nil {
		return nil, err
	}
	paymentRaw, err := s.repo.Get(ctx, domainsettings.KeyPaymentActiveProvider)
	if err != nil {
		return nil, err
	}
	settings := domainsettings.RehydratePlatformSettings(smsRaw, paymentRaw)
	s.store(settings)
	return settings, nil
}

func (s *Service) store(settings *domainsettings.PlatformSettings) {
	s.mu.Lock()
	s.cache = settings
	s.mu.Unlock()
}
