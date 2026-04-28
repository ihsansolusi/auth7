package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
	"github.com/redis/go-redis/v9"
)

type MFAService struct {
	store     *postgres.Store
	redis     *redis.Client
	totp      *TOTPService
	emailOTP  *EmailOTPService
	backup    *BackupCodeService
	encryptor *Encryptor
}

func NewMFAService(store *postgres.Store, redisClient *redis.Client, encryptionKey []byte) (*MFAService, error) {
	const op = "NewMFAService"

	encryptor, err := NewEncryptor(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("%s: encryptor: %w", op, err)
	}

	totpStore := &totpConfigStore{store: store}
	totpRedis := &totpRedisClient{redis: redisClient}
	totpSvc := NewTOTPService(totpStore, encryptor, totpRedis)

	rateLimiter := NewRedisRateLimiter(redisClient)
	emailOTCSvc := NewEmailOTPService(store.EmailOTPCodeRepository, redisClient, rateLimiter)

	backupStore := &backupConfigStore{store: store}
	backupSvc := NewBackupCodeService(backupStore)

	return &MFAService{
		store:     store,
		redis:     redisClient,
		totp:      totpSvc,
		emailOTP:  emailOTCSvc,
		backup:    backupSvc,
		encryptor: encryptor,
	}, nil
}

type totpConfigStore struct {
	store *postgres.Store
}

func (s *totpConfigStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error) {
	return s.store.MFAConfigRepository.GetByUserID(ctx, userID)
}

func (s *totpConfigStore) Create(ctx context.Context, cfg *domain.MFAConfig) error {
	return s.store.MFAConfigRepository.Create(ctx, cfg)
}

func (s *totpConfigStore) Update(ctx context.Context, cfg *domain.MFAConfig) error {
	return s.store.MFAConfigRepository.Update(ctx, cfg)
}

type totpRedisClient struct {
	redis *redis.Client
}

func (r *totpRedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.redis.Get(ctx, key).Result()
}

func (r *totpRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.redis.Set(ctx, key, value, expiration).Err()
}

func (r *totpRedisClient) Del(ctx context.Context, keys ...string) error {
	return r.redis.Del(ctx, keys...).Err()
}

type backupConfigStore struct {
	store *postgres.Store
}

func (s *backupConfigStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error) {
	return s.store.MFAConfigRepository.GetByUserID(ctx, userID)
}

func (s *backupConfigStore) Create(ctx context.Context, cfg *domain.MFAConfig) error {
	return s.store.MFAConfigRepository.Create(ctx, cfg)
}

func (s *backupConfigStore) Update(ctx context.Context, cfg *domain.MFAConfig) error {
	return s.store.MFAConfigRepository.Update(ctx, cfg)
}

type EnrollTOTPInput struct {
	UserID uuid.UUID
}

func (s *MFAService) EnrollTOTP(ctx context.Context, input EnrollTOTPInput) (*EnrollTOTPOutput, error) {
	return s.totp.Enroll(ctx, input.UserID)
}

func (s *MFAService) EnableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	return s.totp.Enable(ctx, userID, code)
}

func (s *MFAService) VerifyTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	return s.totp.Verify(ctx, userID, code)
}

type EnrollEmailOTPInput struct {
	UserID uuid.UUID
	Email  string
}

func (s *MFAService) EnrollEmailOTP(ctx context.Context, input EnrollEmailOTPInput) (string, error) {
	return s.emailOTP.Generate(ctx, GenerateEmailOTPInput{
		UserID:  input.UserID,
		Email:   input.Email,
		Purpose: domain.OTPPurposeEnroll,
	})
}

func (s *MFAService) VerifyEmailOTP(ctx context.Context, input VerifyEmailOTPInput) error {
	return s.emailOTP.Verify(ctx, VerifyEmailOTPInput{
		UserID:  input.UserID,
		Code:    input.Code,
		Purpose: input.Purpose,
	})
}

func (s *MFAService) GenerateBackupCodes(ctx context.Context, userID uuid.UUID) (*GenerateBackupCodesOutput, error) {
	return s.backup.Generate(ctx, userID)
}

type VerifyBackupCodeInput struct {
	UserID uuid.UUID
	Code   string
}

func (s *MFAService) VerifyBackupCode(ctx context.Context, input VerifyBackupCodeInput) error {
	return s.backup.Verify(ctx, input.UserID, input.Code)
}

type GetMFAConfigInput struct {
	UserID uuid.UUID
}

type MFAConfigOutput struct {
	IsTOTPEnabled        bool `json:"is_totp_enabled"`
	IsEmailOTPEnabled    bool `json:"is_email_otp_enabled"`
	IsBackupCodesEnabled bool `json:"is_backup_codes_enabled"`
	BackupCodesRemaining int  `json:"backup_codes_remaining"`
}

func (s *MFAService) GetMFAConfig(ctx context.Context, input GetMFAConfigInput) (*MFAConfigOutput, error) {
	cfg, err := s.store.MFAConfigRepository.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("get mfa config: %w", err)
	}

	remaining, _ := s.backup.Remaining(ctx, input.UserID)

	return &MFAConfigOutput{
		IsTOTPEnabled:        cfg.IsTOTPEnabled,
		IsEmailOTPEnabled:    cfg.IsEmailOTPEnabled,
		IsBackupCodesEnabled: cfg.IsBackupCodesEnabled,
		BackupCodesRemaining: remaining,
	}, nil
}

type StepUpAuthInput struct {
	UserID uuid.UUID
	Method string
	Code   string
}

type StepUpAuthOutput struct {
	Success bool
	Message string
}

func (s *MFAService) StepUpAuth(ctx context.Context, input StepUpAuthInput) (*StepUpAuthOutput, error) {
	const op = "MFAService.StepUpAuth"

	cfg, err := s.store.MFAConfigRepository.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: get config: %w", op, err)
	}

	if !cfg.IsFullyEnabled() {
		return &StepUpAuthOutput{Success: true, Message: "mfa not configured"}, nil
	}

	stepUpKey := fmt.Sprintf("mfa:stepup:%s", input.UserID.String())
	s.redis.Set(ctx, stepUpKey, "1", 15*time.Minute)

	switch input.Method {
	case "totp":
		if err := s.totp.Verify(ctx, input.UserID, input.Code); err != nil {
			return nil, fmt.Errorf("%s: totp verify: %w", op, err)
		}
	case "email_otp":
		if err := s.emailOTP.Verify(ctx, VerifyEmailOTPInput{
			UserID:  input.UserID,
			Code:    input.Code,
			Purpose: domain.OTPPurposeLogin,
		}); err != nil {
			return nil, fmt.Errorf("%s: email otp verify: %w", op, err)
		}
	case "backup":
		if err := s.backup.Verify(ctx, input.UserID, input.Code); err != nil {
			return nil, fmt.Errorf("%s: backup verify: %w", op, err)
		}
	default:
		return nil, fmt.Errorf("%s: unknown method: %s", op, input.Method)
	}

	s.redis.Del(ctx, stepUpKey)

	return &StepUpAuthOutput{Success: true, Message: "step-up successful"}, nil
}

func (s *MFAService) IsStepUpRequired(ctx context.Context, userID uuid.UUID) (bool, error) {
	stepUpKey := fmt.Sprintf("mfa:stepup:%s", userID.String())
	exists, err := s.redis.Exists(ctx, stepUpKey).Result()
	if err != nil {
		return false, fmt.Errorf("check step-up: %w", err)
	}
	return exists > 0, nil
}

type SetupMFAInput struct {
	UserID   uuid.UUID
	Method   string
	Email    string
	TOTPCode string
}

func (s *MFAService) SetupMFA(ctx context.Context, input SetupMFAInput) error {
	const op = "MFAService.SetupMFA"

	cfg, err := s.store.MFAConfigRepository.GetByUserID(ctx, input.UserID)
	if err != nil {
		cfg = &domain.MFAConfig{
			ID:        uuid.Must(uuid.NewV7()),
			UserID:    input.UserID,
			CreatedAt: time.Now(),
		}
	}

	cfg.UpdatedAt = time.Now()

	switch input.Method {
	case "totp":
		if err := s.totp.Enable(ctx, input.UserID, input.TOTPCode); err != nil {
			return fmt.Errorf("%s: enable totp: %w", op, err)
		}
		cfg.IsTOTPEnabled = true
	case "email_otp":
		cfg.IsEmailOTPEnabled = true
	case "backup":
		_, err := s.backup.Generate(ctx, input.UserID)
		if err != nil {
			return fmt.Errorf("%s: generate backup codes: %w", op, err)
		}
		cfg.IsBackupCodesEnabled = true
	}

	now := time.Now()
	cfg.MFAEnabledAt = &now

	if err := s.store.MFAConfigRepository.Update(ctx, cfg); err != nil {
		return fmt.Errorf("%s: update config: %w", op, err)
	}

	user, err := s.store.UserRepository.GetByID(ctx, input.UserID)
	if err == nil {
		user.MFAEnabled = true
		user.MFAMethod = domain.MFAMethod(input.Method)
		user.UpdatedAt = time.Now()
		s.store.UserRepository.Update(ctx, user)
	}

	return nil
}