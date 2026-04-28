package mfa

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type EmailOTPService struct {
	store   EmailOTPStore
	redis   *redis.Client
	limiter  RateLimiter
}

type EmailOTPStore interface {
	Create(ctx context.Context, code *domain.EmailOTPCode) error
	GetByUserIDAndPurpose(ctx context.Context, userID uuid.UUID, purpose string) (*domain.EmailOTPCode, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*domain.EmailOTPCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

func NewEmailOTPService(store EmailOTPStore, redisClient *redis.Client, limiter RateLimiter) *EmailOTPService {
	return &EmailOTPService{
		store:   store,
		redis:   redisClient,
		limiter: limiter,
	}
}

func (s *EmailOTPService) generateCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("generate digit: %w", err)
		}
		code[i] = digits[n.Int64()]
	}
	return string(code), nil
}

type GenerateEmailOTPInput struct {
	UserID  uuid.UUID
	Email   string
	Purpose string
}

func (s *EmailOTPService) Generate(ctx context.Context, input GenerateEmailOTPInput) (string, error) {
	const op = "EmailOTPService.Generate"

	limitKey := fmt.Sprintf("mfa:email_otp:rate:%s", input.UserID.String())
	allowed, err := s.limiter.Allow(ctx, limitKey, 3, time.Hour)
	if err != nil {
		return "", fmt.Errorf("%s: rate limit check: %w", op, err)
	}
	if !allowed {
		return "", fmt.Errorf("%s: rate limit exceeded", op)
	}

	code, err := s.generateCode()
	if err != nil {
		return "", fmt.Errorf("%s: generate code: %w", op, err)
	}

	codeHash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%s: hash code: %w", op, err)
	}

	redisKey := fmt.Sprintf("mfa:email_otp:code:%s:%s", input.UserID.String(), input.Purpose)
	if err := s.redis.Set(ctx, redisKey, string(codeHash), 10*time.Minute).Err(); err != nil {
		return "", fmt.Errorf("%s: cache code: %w", op, err)
	}

	emailCode := &domain.EmailOTPCode{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    input.UserID,
		Code:      string(codeHash),
		Purpose:   input.Purpose,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	if err := s.store.Create(ctx, emailCode); err != nil {
		return "", fmt.Errorf("%s: create code record: %w", op, err)
	}

	return code, nil
}

type VerifyEmailOTPInput struct {
	UserID  uuid.UUID
	Code    string
	Purpose string
}

func (s *EmailOTPService) Verify(ctx context.Context, input VerifyEmailOTPInput) error {
	const op = "EmailOTPService.Verify"

	redisKey := fmt.Sprintf("mfa:email_otp:code:%s:%s", input.UserID.String(), input.Purpose)
	storedHash, err := s.redis.Get(ctx, redisKey).Result()
	if err != nil {
		return fmt.Errorf("%s: get cached code: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(input.Code)); err != nil {
		return fmt.Errorf("%s: invalid code", op)
	}

	s.redis.Del(ctx, redisKey)

	latestCode, err := s.store.GetActiveByUserID(ctx, input.UserID)
	if err == nil && latestCode != nil {
		s.store.MarkUsed(ctx, latestCode.ID)
	}

	return nil
}

func (s *EmailOTPService) CleanupExpired(ctx context.Context, userID uuid.UUID) error {
	return s.store.DeleteByUserID(ctx, userID)
}