package events

import (
	"context"
	"encoding/json"
	"fmt"

	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
)

// OutboxPublisher, domain event'lerini aynı DB transaction'ında outbox_jobs'a yazar.
type OutboxPublisher struct {
	repo appoutbox.Enqueuer
}

// NewOutboxPublisher, publisher'ı kurar.
func NewOutboxPublisher(repo appoutbox.Enqueuer) *OutboxPublisher {
	return &OutboxPublisher{repo: repo}
}

// Publish, event'leri kalıcı outbox'a ekler.
func (p *OutboxPublisher) Publish(ctx context.Context, events ...shared.DomainEvent) error {
	actor := appshared.ActorFromContext(ctx)
	if actor.ActorID == "" {
		actor.ActorID = logger.UserID(ctx)
	}
	if actor.IP == "" {
		actor.IP = logger.ClientIP(ctx)
	}
	if actor.UserAgent == "" {
		actor.UserAgent = logger.UserAgent(ctx)
	}
	if actor.CorrelationID == "" {
		actor.CorrelationID = logger.CorrelationID(ctx)
	}

	for _, e := range events {
		if e == nil {
			continue
		}
		eventID := e.EventID()
		if eventID == "" {
			return fmt.Errorf("events: boş event id (%s)", e.EventName())
		}
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("events: marshal %s: %w", e.EventName(), err)
		}
		aggType := appoutbox.AggregateTypeFromEvent(e.EventName())
		payload := appoutbox.DomainEventPayload{
			EventID:       eventID,
			EventName:     e.EventName(),
			AggregateType: aggType,
			AggregateID:   e.AggregateID(),
			OccurredAt:    e.OccurredAt(),
			ActorID:       actor.ActorID,
			ActorType:     actor.ActorType,
			ActorEmail:    actor.ActorEmail,
			Source:        actor.Source,
			CorrelationID: actor.CorrelationID,
			IP:            actor.IP,
			UserAgent:     actor.UserAgent,
			Data:          data,
		}
		job, err := appoutbox.NewJob(
			appoutbox.KindDomainEvent,
			aggType,
			e.AggregateID(),
			eventID,
			payload,
		)
		if err != nil {
			return err
		}
		job.ID = eventID
		if err := p.repo.Enqueue(ctx, job); err != nil {
			return fmt.Errorf("events: outbox enqueue %s: %w", e.EventName(), err)
		}
	}
	return nil
}
