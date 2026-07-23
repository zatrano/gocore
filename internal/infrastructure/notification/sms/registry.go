package sms

import (
	"context"
	"fmt"
	"log/slog"

	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
)

// activeReader, aktif sağlayıcı adını çözer.
type activeReader interface {
	ActiveProvider(ctx context.Context) (string, error)
}

// Registry, kayıtlı SMS sağlayıcıları arasından dashboard'da seçilen aktif
// olanı kullanarak gönderim yapar.
type Registry struct {
	providers map[string]Provider
	active    activeReader
	log       *slog.Logger
}

// NewRegistry, tüm sağlayıcıları registry ile sarar.
func NewRegistry(providers []Provider, active activeReader, log *slog.Logger) *Registry {
	m := make(map[string]Provider, len(providers))
	for _, p := range providers {
		m[p.Name()] = p
	}
	return &Registry{providers: m, active: active, log: log}
}

func (r *Registry) Name() string { return "registry" }

// Send, aktif sağlayıcı üzerinden SMS gönderir.
func (r *Registry) Send(ctx context.Context, to, body string) error {
	name, err := r.active.ActiveProvider(ctx)
	if err != nil {
		r.log.WarnContext(ctx, "sms aktif sağlayıcı okunamadı, netgsm kullanılıyor",
			slog.String("error", err.Error()))
		name = domainsettings.ProviderNetgsm.String()
	}
	p, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("sms: kayıtlı olmayan sağlayıcı %q", name)
	}
	return p.Send(ctx, to, body)
}

// RegisteredNames, kayıtlı sağlayıcı adlarını döner.
func (r *Registry) RegisteredNames() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
