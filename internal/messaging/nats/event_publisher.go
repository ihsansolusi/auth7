package nats

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

type EventPublisher struct {
	publisher *Publisher
	logger    zerolog.Logger
}

func NewEventPublisher(publisher *Publisher, logger zerolog.Logger) *EventPublisher {
	return &EventPublisher{
		publisher: publisher,
		logger:    logger,
	}
}

// PublishAudit durably publishes a pre-marshalled audit event to JetStream
// (used by the audit forwarder to reach audit7's AUDIT7_EVENTS stream). msgID
// sets Nats-Msg-Id for dedup. Returns an error if the persist ack fails.
func (ep *EventPublisher) PublishAudit(subject string, data []byte, msgID string) error {
	return ep.publisher.client.PublishStream(subject, data, msgID)
}

func (ep *EventPublisher) PublishTokenRevoked(ctx context.Context, event TokenRevokedEvent) error {
	const op = "messaging.PublishTokenRevoked"
	if err := ep.publisher.Publish(ctx, SubjectTokenRevoked, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishTokenRefreshed(ctx context.Context, event TokenRefreshedEvent) error {
	const op = "messaging.PublishTokenRefreshed"
	if err := ep.publisher.Publish(ctx, SubjectTokenRefreshed, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishSessionCreated(ctx context.Context, event SessionCreatedEvent) error {
	const op = "messaging.PublishSessionCreated"
	if err := ep.publisher.Publish(ctx, SubjectSessionCreated, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishSessionTerminated(ctx context.Context, event SessionTerminatedEvent) error {
	const op = "messaging.PublishSessionTerminated"
	if err := ep.publisher.Publish(ctx, SubjectSessionTerminated, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishSessionRevokedAll(ctx context.Context, event SessionRevokedAllEvent) error {
	const op = "messaging.PublishSessionRevokedAll"
	if err := ep.publisher.Publish(ctx, SubjectSessionRevokedAll, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishBranchSwitched(ctx context.Context, event BranchSwitchedEvent) error {
	const op = "messaging.PublishBranchSwitched"
	if err := ep.publisher.Publish(ctx, SubjectBranchSwitched, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) PublishSecurityAlert(ctx context.Context, event SecurityAlertEvent) error {
	const op = "messaging.PublishSecurityAlert"
	if err := ep.publisher.Publish(ctx, SubjectSecurityAlert, event); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (ep *EventPublisher) Close() {
	ep.publisher.client.Close()
}

func (ep *EventPublisher) IsConnected() bool {
	return ep.publisher.client.IsConnected()
}
