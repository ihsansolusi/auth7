package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/pquerna/otp/totp"
)

type TOTPService struct {
	store     TOTPStore
	encryptor *Encryptor
	redis     TOTPRedisClient
}

type TOTPStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error)
	Create(ctx context.Context, cfg *domain.MFAConfig) error
	Update(ctx context.Context, cfg *domain.MFAConfig) error
}

type TOTPRedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

func NewTOTPService(store TOTPStore, encryptor *Encryptor, redis TOTPRedisClient) *TOTPService {
	return &TOTPService{
		store:     store,
		encryptor: encryptor,
		redis:     redis,
	}
}

type EnrollTOTPOutput struct {
	Secret     string
	QRCodeData string
}

func (s *TOTPService) Enroll(ctx context.Context, userID uuid.UUID) (*EnrollTOTPOutput, error) {
	const op = "TOTPService.Enroll"

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Auth7",
		AccountName: userID.String(),
		Secret:      make([]byte, 20),
	})
	if err != nil {
		return nil, fmt.Errorf("%s: generate key: %w", op, err)
	}

	secretBase32 := key.Secret()
	qrData := key.URL()

	cfg, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: get config: %w", op, err)
	}
	if cfg == nil {
		cfg = &domain.MFAConfig{
			ID:        uuid.Must(uuid.NewV7()),
			UserID:    userID,
			CreatedAt: time.Now(),
		}
	}

	encryptedSecret, iv, err := s.encryptor.Encrypt([]byte(secretBase32))
	if err != nil {
		return nil, fmt.Errorf("%s: encrypt secret: %w", op, err)
	}

	cfg.TOTPSecretEncrypted = encryptedSecret
	cfg.TOTPSecretIV = iv
	cfg.IsTOTPEnabled = false
	cfg.UpdatedAt = time.Now()

	if cfg.ID == uuid.Nil {
		if err := s.store.Create(ctx, cfg); err != nil {
			return nil, fmt.Errorf("%s: create config: %w", op, err)
		}
	} else {
		if err := s.store.Update(ctx, cfg); err != nil {
			return nil, fmt.Errorf("%s: update config: %w", op, err)
		}
	}

	return &EnrollTOTPOutput{
		Secret:     secretBase32,
		QRCodeData: qrData,
	}, nil
}

func (s *TOTPService) Enable(ctx context.Context, userID uuid.UUID, code string) error {
	const op = "TOTPService.Enable"

	cfg, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: get config: %w", op, err)
	}
	if cfg == nil {
		return fmt.Errorf("%s: mfa config not found", op)
	}

	if len(cfg.TOTPSecretEncrypted) == 0 {
		return fmt.Errorf("%s: totp not enrolled", op)
	}

	secretBytes, err := s.encryptor.Decrypt(cfg.TOTPSecretEncrypted, cfg.TOTPSecretIV)
	if err != nil {
		return fmt.Errorf("%s: decrypt secret: %w", op, err)
	}

	if !totp.Validate(code, string(secretBytes)) {
		return fmt.Errorf("%s: invalid code", op)
	}

	replayKey := fmt.Sprintf("mfa:totp:replay:%s:%s", userID.String(), code)
	exists, err := s.redis.Get(ctx, replayKey)
	if err == nil && exists != "" {
		return fmt.Errorf("%s: code already used", op)
	}

	if err := s.redis.Set(ctx, replayKey, "1", 30*time.Second); err != nil {
		return fmt.Errorf("%s: set replay key: %w", op, err)
	}

	cfg.IsTOTPEnabled = true
	now := time.Now()
	cfg.MFAEnabledAt = &now
	cfg.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, cfg); err != nil {
		return fmt.Errorf("%s: update config: %w", op, err)
	}

	return nil
}

func (s *TOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) error {
	const op = "TOTPService.Verify"

	cfg, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: get config: %w", op, err)
	}
	if cfg == nil {
		return fmt.Errorf("%s: mfa config not found", op)
	}

	if !cfg.IsTOTPEnabled || len(cfg.TOTPSecretEncrypted) == 0 {
		return fmt.Errorf("%s: totp not enabled", op)
	}

	secretBytes, err := s.encryptor.Decrypt(cfg.TOTPSecretEncrypted, cfg.TOTPSecretIV)
	if err != nil {
		return fmt.Errorf("%s: decrypt secret: %w", op, err)
	}

	if !totp.Validate(code, string(secretBytes)) {
		return fmt.Errorf("%s: invalid code", op)
	}

	replayKey := fmt.Sprintf("mfa:totp:replay:%s:%s", userID.String(), code)
	exists, err := s.redis.Get(ctx, replayKey)
	if err == nil && exists != "" {
		return fmt.Errorf("%s: code already used", op)
	}

	if err := s.redis.Set(ctx, replayKey, "1", 30*time.Second); err != nil {
		return fmt.Errorf("%s: set replay key: %w", op, err)
	}

	return nil
}

func GenerateTOTPCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

func ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}