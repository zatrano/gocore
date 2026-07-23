package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
)

// SideEffect, domain event outbox işi için ikincil etki (ör. e-posta).
type SideEffect func(ctx context.Context, p appoutbox.DomainEventPayload) error

// DomainEventHandler, kind=domain.event işlerini audit + yan etkilere yönlendirir.
func DomainEventHandler(auditor appshared.AuditLogger, log *slog.Logger, effects map[string][]SideEffect) Handler {
	if effects == nil {
		effects = map[string][]SideEffect{}
	}
	return func(ctx context.Context, job appoutbox.Job) error {
		var p appoutbox.DomainEventPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("domain event payload: %w", err)
		}
		resource := p.AggregateType
		if resource == "" {
			resource = appoutbox.ResourceFromEvent(p.EventName)
		}
		entry := appshared.AuditEntry{
			EventID:       p.EventID,
			OccurredAt:    p.OccurredAt,
			ActorID:       p.ActorID,
			ActorType:     p.ActorType,
			ActorEmail:    p.ActorEmail,
			Action:        p.EventName,
			Resource:      resource,
			ResourceID:    p.AggregateID,
			Source:        p.Source,
			CorrelationID: p.CorrelationID,
			IP:            p.IP,
			UserAgent:     p.UserAgent,
			Metadata:      appaudit.MetadataFromDomainEvent(p),
		}
		if err := auditor.Log(ctx, entry); err != nil {
			return fmt.Errorf("audit log: %w", err)
		}
		for _, fx := range effects[p.EventName] {
			if err := fx(ctx, p); err != nil {
				log.WarnContext(ctx, "domain event side effect failed",
					slog.String("event", p.EventName),
					slog.String("error", err.Error()),
				)
				return err
			}
		}
		return nil
	}
}
