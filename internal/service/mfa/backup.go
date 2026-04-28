package mfa

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type BackupCodeService struct {
	store BackupCodeStore
}

type BackupCodeStore interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error)
	Create(ctx context.Context, cfg *domain.MFAConfig) error
	Update(ctx context.Context, cfg *domain.MFAConfig) error
}

func NewBackupCodeService(store BackupCodeStore) *BackupCodeService {
	return &BackupCodeService{store: store}
}

func (s *BackupCodeService) generateCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 10)
	if _, err := rand.Read(code); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	for i := range code {
		code[i] = chars[int(code[i])%len(chars)]
	}
	return string(code), nil
}

type GenerateBackupCodesOutput struct {
	Codes []string
}

func (s *BackupCodeService) Generate(ctx context.Context, userID uuid.UUID) (*GenerateBackupCodesOutput, error) {
	const op = "BackupCodeService.Generate"

	codes := make([]string, 10)
	hashes := make([]string, 10)

	for i := 0; i < 10; i++ {
		code, err := s.generateCode()
		if err != nil {
			return nil, fmt.Errorf("%s: generate code: %w", op, err)
		}
		codes[i] = code

		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("%s: hash code: %w", op, err)
		}
		hashes[i] = string(hash)
	}

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

	cfg.BackupCodesHash = hashes
	cfg.IsBackupCodesEnabled = true
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

	return &GenerateBackupCodesOutput{Codes: codes}, nil
}

func (s *BackupCodeService) Verify(ctx context.Context, userID uuid.UUID, code string) error {
	const op = "BackupCodeService.Verify"

	cfg, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: get config: %w", op, err)
	}
	if cfg == nil {
		return fmt.Errorf("%s: mfa config not found", op)
	}

	if !cfg.IsBackupCodesEnabled || len(cfg.BackupCodesHash) == 0 {
		return fmt.Errorf("%s: backup codes not enabled", op)
	}

	code = strings.ToUpper(strings.TrimSpace(code))

	found := false
	for i, hash := range cfg.BackupCodesHash {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil {
			found = true
			cfg.BackupCodesHash = append(cfg.BackupCodesHash[:i], cfg.BackupCodesHash[i+1:]...)
			break
		}
	}

	if !found {
		return fmt.Errorf("%s: invalid backup code", op)
	}

	cfg.UpdatedAt = time.Now()
	if err := s.store.Update(ctx, cfg); err != nil {
		return fmt.Errorf("%s: update config: %w", op, err)
	}

	return nil
}

func (s *BackupCodeService) Remaining(ctx context.Context, userID uuid.UUID) (int, error) {
	cfg, err := s.store.GetByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get config: %w", err)
	}
	return len(cfg.BackupCodesHash), nil
}