// Package notif7client provides a lightweight HTTP client for sending events
// to the notif7 notification service.
package notif7client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender is the interface implemented by Client and NoopSender.
type Sender interface {
	Send(ctx context.Context, event Event) (*SendResult, error)
}

// Compile-time interface checks.
var _ Sender = (*Client)(nil)
var _ Sender = NoopSender{}

// Client sends notification events to a notif7 instance.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New constructs a Client. baseURL is the notif7 base URL (e.g. "http://notif7:8082").
// apiKey is a producer JWT signed with the notif7 API key secret.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// NoopSender discards all events. Use in development or test environments
// where notif7 is not available.
type NoopSender struct{}

// Send implements Sender by doing nothing and returning nil.
func (NoopSender) Send(_ context.Context, _ Event) (*SendResult, error) { return nil, nil }

// Event is the notification payload sent by a producer.
type Event struct {
	Source           string         `json:"source"`
	EventType        string         `json:"event_type"`
	UserIDs          []string       `json:"user_ids"`
	Title            string         `json:"title"`
	Body             string         `json:"body,omitempty"`
	Payload          map[string]any `json:"payload,omitempty"`
	RefID            string         `json:"ref_id,omitempty"`
	RefURL           string         `json:"ref_url,omitempty"`
	DeliveryChannels []string       `json:"delivery_channels,omitempty"` // ["in_app"] | ["in_app","email"]
	EmailAddresses   []string       `json:"email_addresses,omitempty"`   // 1:1 with UserIDs, required for email
}

// SendResult is the response from a successful Send call.
type SendResult struct {
	Accepted int      `json:"accepted"`
	EventIDs []string `json:"event_ids"`
}

// Send dispatches an event to notif7. Safe to call in a goroutine (fire-and-forget).
// Returns an error if notif7 is unreachable or returns a non-202 status.
func (c *Client) Send(ctx context.Context, event Event) (*SendResult, error) {
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("notif7client: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/events", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("notif7client: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("notif7client: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusAccepted {
		var result SendResult
		_ = json.NewDecoder(resp.Body).Decode(&result)
		return &result, nil
	}

	var errBody struct {
		Error map[string]string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	msg := errBody.Error["message"]
	if msg == "" {
		msg = "unknown error"
	}
	return nil, fmt.Errorf("notif7client: status %d: %s", resp.StatusCode, msg)
}
