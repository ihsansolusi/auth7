package notif7

import (
	"context"
	"fmt"
	"time"

	"github.com/ihsansolusi/lib7-service-go/notif7client"
)

type Client struct {
	sender notif7client.Sender
	logger any
}

func NewClient(sender notif7client.Sender) *Client {
	return &Client{
		sender: sender,
	}
}

type LoginNewDeviceParams struct {
	UserID    string
	Username  string
	Email     string
	OrgID     string
	DeviceName string
	IPAddress string
	Location  string
}

func (c *Client) SendLoginNewDevice(ctx context.Context, params LoginNewDeviceParams) error {
	const op = "notif7.SendLoginNewDevice"

	event := notif7client.Event{
		Source:    "auth7",
		EventType: "auth.login_new_device",
		UserIDs:   []string{params.UserID},
		Title:     "New login detected",
		Body:      fmt.Sprintf("New login to your account on %s from IP %s (%s)", params.DeviceName, params.IPAddress, params.Location),
		Payload: map[string]any{
			"username":    params.Username,
			"email":       params.Email,
			"org_id":      params.OrgID,
			"device_name": params.DeviceName,
			"ip_address":  params.IPAddress,
			"location":    params.Location,
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type AccountLockedParams struct {
	UserID   string
	Username string
	Email    string
	OrgID    string
	Reason   string
	LockedAt time.Time
}

func (c *Client) SendAccountLocked(ctx context.Context, params AccountLockedParams) error {
	const op = "notif7.SendAccountLocked"

	event := notif7client.Event{
		Source:    "auth7",
		EventType: "auth.account_locked",
		UserIDs:   []string{params.UserID},
		Title:     "Account locked",
		Body:      fmt.Sprintf("Your account has been locked due to: %s", params.Reason),
		Payload: map[string]any{
			"username":  params.Username,
			"email":     params.Email,
			"org_id":    params.OrgID,
			"reason":    params.Reason,
			"locked_at": params.LockedAt.Format(time.RFC3339),
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type PasswordChangedParams struct {
	UserID    string
	Username  string
	Email     string
	OrgID     string
	ChangedAt time.Time
	IPAddress string
}

func (c *Client) SendPasswordChanged(ctx context.Context, params PasswordChangedParams) error {
	const op = "notif7.SendPasswordChanged"

	event := notif7client.Event{
		Source:    "auth7",
		EventType: "auth.password_changed",
		UserIDs:   []string{params.UserID},
		Title:     "Password changed",
		Body:      "Your password was changed. If this wasn't you, please contact support immediately.",
		Payload: map[string]any{
			"username":    params.Username,
			"email":       params.Email,
			"org_id":      params.OrgID,
			"changed_at":  params.ChangedAt.Format(time.RFC3339),
			"ip_address":  params.IPAddress,
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

type MfaResetParams struct {
	UserID    string
	Username  string
	Email     string
	OrgID     string
	ResetAt   time.Time
	IPAddress string
}

func (c *Client) SendMfaReset(ctx context.Context, params MfaResetParams) error {
	const op = "notif7.SendMfaReset"

	event := notif7client.Event{
		Source:    "auth7",
		EventType: "auth.mfa_reset",
		UserIDs:   []string{params.UserID},
		Title:     "MFA reset",
		Body:      "Your MFA settings have been reset by an administrator.",
		Payload: map[string]any{
			"username":   params.Username,
			"email":      params.Email,
			"org_id":     params.OrgID,
			"reset_at":   params.ResetAt.Format(time.RFC3339),
			"ip_address": params.IPAddress,
		},
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.sender.Send(ctx, event)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

var _ Interface = (*Client)(nil)

type Interface interface {
	SendLoginNewDevice(ctx context.Context, params LoginNewDeviceParams) error
	SendAccountLocked(ctx context.Context, params AccountLockedParams) error
	SendPasswordChanged(ctx context.Context, params PasswordChangedParams) error
	SendMfaReset(ctx context.Context, params MfaResetParams) error
}
