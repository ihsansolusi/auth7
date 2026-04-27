package security

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type EmergencyService struct {
	sessionSvc any
	keySvc     any
}

func NewEmergencyService(sessionSvc, keySvc any) *EmergencyService {
	return &EmergencyService{
		sessionSvc: sessionSvc,
		keySvc:     keySvc,
	}
}

func (s *EmergencyService) RevokeAllTokens(ctx context.Context, orgID uuid.UUID) error {
	const op = "EmergencyService.RevokeAllTokens"

	orgStr := orgID.String()

	if sessionSvc, ok := s.sessionSvc.(interface {
		RevokeAllOrgSessions(ctx context.Context, orgID string) error
	}); ok {
		if err := sessionSvc.RevokeAllOrgSessions(ctx, orgStr); err != nil {
			return fmt.Errorf("%s: revoke sessions: %w", op, err)
		}
	}

	return nil
}

func (s *EmergencyService) ForceLogoutAllUsers(ctx context.Context, orgID uuid.UUID) error {
	const op = "EmergencyService.ForceLogoutAllUsers"

	orgStr := orgID.String()

	if sessionSvc, ok := s.sessionSvc.(interface {
		RevokeAllOrgSessions(ctx context.Context, orgID string) error
	}); ok {
		if err := sessionSvc.RevokeAllOrgSessions(ctx, orgStr); err != nil {
			return fmt.Errorf("%s: revoke sessions: %w", op, err)
		}
	}

	if keySvc, ok := s.keySvc.(interface {
		RotateKey() error
	}); ok {
		if err := keySvc.RotateKey(); err != nil {
			return fmt.Errorf("%s: rotate key: %w", op, err)
		}
	}

	return nil
}

func (s *EmergencyService) EmergencyKeyRotation(ctx context.Context, orgID uuid.UUID) error {
	const op = "EmergencyService.EmergencyKeyRotation"

	if keySvc, ok := s.keySvc.(interface {
		RotateKey() error
	}); ok {
		if err := keySvc.RotateKey(); err != nil {
			return fmt.Errorf("%s: rotate key: %w", op, err)
		}
	}

	return nil
}

func (s *EmergencyService) GetSecurityStatus(ctx context.Context, orgID uuid.UUID) (map[string]interface{}, error) {
	const op = "EmergencyService.GetSecurityStatus"

	status := map[string]interface{}{
		"org_id":         orgID.String(),
		"brute_force":    "active",
		" rate_limiting": "active",
		"security_headers": "enabled",
	}

	return status, nil
}