// Package events, domain event yayınlama portunun implementasyonlarını içerir.
// Varsayılan implementasyon in-memory dispatcher'dır; üretimde Kafka/RabbitMQ
// adaptörüyle değiştirilebilir (appshared.EventPublisher portu sayesinde).
package events

import (
	"context"
	"log/slog"
	"sync"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// Handler, belirli bir event'i işleyen fonksiyondur.
type Handler func(ctx context.Context, e shared.DomainEvent) error

// InMemoryPublisher, event'leri kayıtlı handler'lara senkron dağıtır ve loglar.
// Basit, bağımlılıksız ve test edilebilir; event-driven mimariye giriş noktasıdır.
// Kafka/RabbitMQ için bu tipi değiştirmek yeterlidir (arayüz aynı kalır).
type InMemoryPublisher struct {
	log      *slog.Logger
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewInMemoryPublisher, publisher'ı kurar.
func NewInMemoryPublisher(log *slog.Logger) *InMemoryPublisher {
	return &InMemoryPublisher{log: log, handlers: make(map[string][]Handler)}
}

// Subscribe, bir event adına handler kaydeder.
func (p *InMemoryPublisher) Subscribe(eventName string, h Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[eventName] = append(p.handlers[eventName], h)
}

// Publish, verilen event'leri loglar ve ilgili handler'lara dağıtır.
// Handler hataları loglanır ama akışı kesmez (at-least-once yerine best-effort;
// güçlü garanti gerekiyorsa transactional outbox pattern eklenmelidir).
func (p *InMemoryPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, e := range events {
		p.log.InfoContext(ctx, "domain event",
			slog.String("event", e.EventName()),
			slog.String("aggregate_id", e.AggregateID()),
		)
		for _, h := range p.handlers[e.EventName()] {
			if err := h(ctx, e); err != nil {
				p.log.ErrorContext(ctx, "event handler failed",
					slog.String("event", e.EventName()),
					slog.String("error", err.Error()),
				)
			}
		}
	}
	return nil
}
