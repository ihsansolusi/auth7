package tests

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	natsmessaging "github.com/ihsansolusi/auth7/internal/messaging/nats"
)

func skipIfNATSUnavailable(t *testing.T) *nats.Conn {
	t.Helper()

	nc, err := nats.Connect("nats://localhost:4222", nats.Timeout(2*time.Second))
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	return nc
}

func TestNATSClient_Connect(t *testing.T) {
	nc := skipIfNATSUnavailable(t)
	defer nc.Close()

	assert.True(t, nc.IsConnected())
}

func TestNATS_PublishSubscribe(t *testing.T) {
	nc := skipIfNATSUnavailable(t)
	defer nc.Close()

	received := make(chan []byte, 1)

	sub, err := nc.Subscribe("test.auth7.subject", func(msg *nats.Msg) {
		received <- msg.Data
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	testData := []byte(`{"test": "data"}`)
	err = nc.Publish("test.auth7.subject", testData)
	require.NoError(t, err)

	select {
	case data := <-received:
		assert.Equal(t, testData, data)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestNATSClient_NewClient(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4222",
		Name:           "auth7-test",
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  3,
		PublishTimeout: 5 * time.Second,
		PublishRetry:   3,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	defer client.Close()

	require.NotNil(t, client)
	assert.True(t, client.Conn().IsConnected())
}

func TestNATSClient_PublishViaPublisher(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4222",
		Name:           "auth7-test",
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  3,
		PublishTimeout: 5 * time.Second,
		PublishRetry:   3,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	defer client.Close()

	publisher := natsmessaging.NewPublisher(client, logger)

	err = publisher.Publish(context.Background(), "test.auth7.publish", map[string]string{"msg": "hello"})
	assert.NoError(t, err)
}

func TestEventPublisher_PublishTokenRevoked(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4222",
		Name:           "auth7-test",
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  3,
		PublishTimeout: 5 * time.Second,
		PublishRetry:   3,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	defer client.Close()

	publisher := natsmessaging.NewPublisher(client, logger)
	eventPub := natsmessaging.NewEventPublisher(publisher, logger)

	err = eventPub.PublishTokenRevoked(context.Background(), natsmessaging.TokenRevokedEvent{
		TokenID:   "tok-123",
		OrgID:     "org-456",
		UserID:    "user-789",
		RevokedBy: "admin",
		Reason:    "manual_revocation",
		RevokedAt: time.Now(),
	})
	assert.NoError(t, err)
}

func TestEventPublisher_PublishSessionCreated(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4222",
		Name:           "auth7-test",
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  3,
		PublishTimeout: 5 * time.Second,
		PublishRetry:   3,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	defer client.Close()

	publisher := natsmessaging.NewPublisher(client, logger)
	eventPub := natsmessaging.NewEventPublisher(publisher, logger)

	err = eventPub.PublishSessionCreated(context.Background(), natsmessaging.SessionCreatedEvent{
		SessionID: "sess-123",
		OrgID:     "org-456",
		UserID:    "user-789",
		IPAddress: "192.168.1.1",
		CreatedAt: time.Now(),
	})
	assert.NoError(t, err)
}

func TestEventPublisher_PublishSecurityAlert(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4222",
		Name:           "auth7-test",
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  3,
		PublishTimeout: 5 * time.Second,
		PublishRetry:   3,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS not available, skipping test")
	}
	defer client.Close()

	publisher := natsmessaging.NewPublisher(client, logger)
	eventPub := natsmessaging.NewEventPublisher(publisher, logger)

	err = eventPub.PublishSecurityAlert(context.Background(), natsmessaging.SecurityAlertEvent{
		Type:      natsmessaging.AlertBruteForce,
		OrgID:     "org-456",
		UserID:    "user-789",
		IPAddress: "10.0.0.1",
		Details:   map[string]any{"attempts": 5},
		AlertedAt: time.Now(),
	})
	assert.NoError(t, err)
}

func TestNATS_EventSubjects(t *testing.T) {
	assert.Equal(t, "auth7.tokens.revoked", natsmessaging.SubjectTokenRevoked)
	assert.Equal(t, "auth7.tokens.refreshed", natsmessaging.SubjectTokenRefreshed)
	assert.Equal(t, "auth7.sessions.created", natsmessaging.SubjectSessionCreated)
	assert.Equal(t, "auth7.sessions.terminated", natsmessaging.SubjectSessionTerminated)
	assert.Equal(t, "auth7.sessions.revoked_all", natsmessaging.SubjectSessionRevokedAll)
	assert.Equal(t, "auth7.security.alert", natsmessaging.SubjectSecurityAlert)
}

func TestNATS_SecurityAlertTypes(t *testing.T) {
	assert.Equal(t, natsmessaging.SecurityAlertType("brute_force"), natsmessaging.AlertBruteForce)
	assert.Equal(t, natsmessaging.SecurityAlertType("new_device"), natsmessaging.AlertNewDevice)
	assert.Equal(t, natsmessaging.SecurityAlertType("ip_change"), natsmessaging.AlertIPChange)
	assert.Equal(t, natsmessaging.SecurityAlertType("suspicious_login"), natsmessaging.AlertSuspiciousLogin)
	assert.Equal(t, natsmessaging.SecurityAlertType("account_locked"), natsmessaging.AlertAccountLocked)
}

func TestNATS_PublishFailWhenDisconnected(t *testing.T) {
	logger := zerolog.Nop()

	cfg := natsmessaging.Config{
		URL:            "nats://localhost:4223",
		Name:           "auth7-test",
		ReconnectWait:  100 * time.Millisecond,
		MaxReconnects:  1,
		PublishTimeout: 1 * time.Second,
		PublishRetry:   1,
	}

	client, err := natsmessaging.NewClient(context.Background(), cfg, logger)
	if err != nil {
		t.Skip("NATS unexpectedly available on wrong port")
	}
	defer client.Close()

	publisher := natsmessaging.NewPublisher(client, logger)
	eventPub := natsmessaging.NewEventPublisher(publisher, logger)

	err = eventPub.PublishTokenRevoked(context.Background(), natsmessaging.TokenRevokedEvent{
		TokenID:   "tok-123",
		OrgID:     "org-456",
		UserID:    "user-789",
		RevokedBy: "admin",
		Reason:    "test",
		RevokedAt: time.Now(),
	})

	assert.Error(t, err, "publish should fail when NATS is unavailable")
}
