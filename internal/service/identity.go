package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/ihsansolusi/auth7/internal/store/postgres"
)

type IdentityService struct {
	store     *postgres.Store
	hasher    *password.Hasher
	tracer    any
	logger    any
}

func NewIdentityService(store *postgres.Store, hasher *password.Hasher) *IdentityService {
	return &IdentityService{
		store:  store,
		hasher: hasher,
	}
}

type RegisterInput struct {
	OrgID     uuid.UUID
	Username  string
	Email     string
	Password  string
	FullName  string
}

type RegisterOutput struct {
	User         *domain.User
	VerifyToken string
}

func (s *IdentityService) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	const op = "IdentityService.Register"

	email := domain.NormalizeEmail(input.Email)

	existingUser, err := s.store.UserRepository.GetByUsername(ctx, input.OrgID, input.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("%s: username already exists", op)
	}

	existingUser, err = s.store.UserRepository.GetByEmail(ctx, input.OrgID, email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("%s: email already exists", op)
	}

	if err := domain.DefaultPasswordPolicy.Validate(input.Password, input.Username, email); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("%s: hash password: %w", op, err)
	}

	now := time.Now()
	user := &domain.User{
		ID:                     uuid.Must(uuid.NewV7()),
		OrgID:                  input.OrgID,
		Username:               input.Username,
		Email:                  email,
		FullName:               input.FullName,
		Status:                 domain.UserStatusPendingVerification,
		EmailVerified:          false,
		MFAEnabled:             false,
		RequirePasswordChange:   false,
		FailedLoginAttempts:     0,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("%s: validation: %w", op, err)
	}

	if err := s.store.UserRepository.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("%s: create user: %w", op, err)
	}

	cred := &domain.UserCredential{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		CredentialType: domain.CredentialTypePassword,
		SecretHash:     passwordHash,
		Version:        1,
		IsCurrent:      true,
		CreatedAt:      now,
	}

	if err := s.store.CredentialRepository.Create(ctx, cred); err != nil {
		return nil, fmt.Errorf("%s: create credential: %w", op, err)
	}

	verifyToken := uuid.New().String()
	vt := &domain.VerificationToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		Token:     verifyToken,
		TokenType: domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.store.VerificationTokenRepository.Create(ctx, vt); err != nil {
		return nil, fmt.Errorf("%s: create verify token: %w", op, err)
	}

	return &RegisterOutput{
		User:         user,
		VerifyToken: verifyToken,
	}, nil
}

type VerifyEmailInput struct {
	Token string
}

func (s *IdentityService) VerifyEmail(ctx context.Context, input VerifyEmailInput) error {
	const op = "IdentityService.VerifyEmail"

	vt, err := s.store.VerificationTokenRepository.GetByToken(ctx, input.Token)
	if err != nil {
		return fmt.Errorf("%s: get token: %w", op, err)
	}

	if !vt.IsValid() {
		return fmt.Errorf("%s: token invalid or expired", op)
	}

	user, err := s.store.UserRepository.GetByID(ctx, vt.UserID)
	if err != nil {
		return fmt.Errorf("%s: get user: %w", op, err)
	}

	user.Status = domain.UserStatusActive
	user.EmailVerified = true
	user.UpdatedAt = time.Now()

	if err := s.store.UserRepository.Update(ctx, user); err != nil {
		return fmt.Errorf("%s: update user: %w", op, err)
	}

	if err := s.store.VerificationTokenRepository.MarkUsed(ctx, vt.ID); err != nil {
		return fmt.Errorf("%s: mark token used: %w", op, err)
	}

	return nil
}

type RecoverPasswordInput struct {
	Email string
	OrgID uuid.UUID
}

func (s *IdentityService) RecoverPassword(ctx context.Context, input RecoverPasswordInput) error {
	const op = "IdentityService.RecoverPassword"

	email := domain.NormalizeEmail(input.Email)

	user, err := s.store.UserRepository.GetByEmail(ctx, input.OrgID, email)
	if err != nil {
		return nil
	}

	token := uuid.New().String()
	vt := &domain.VerificationToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		Token:     token,
		TokenType: domain.TokenTypePasswordRecovery,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.store.VerificationTokenRepository.Create(ctx, vt); err != nil {
		return fmt.Errorf("%s: create recovery token: %w", op, err)
	}

	return nil
}

type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

func (s *IdentityService) ResetPassword(ctx context.Context, input ResetPasswordInput) error {
	const op = "IdentityService.ResetPassword"

	vt, err := s.store.VerificationTokenRepository.GetByToken(ctx, input.Token)
	if err != nil {
		return fmt.Errorf("%s: get token: %w", op, err)
	}

	if !vt.IsValid() {
		return fmt.Errorf("%s: token invalid or expired", op)
	}

	user, err := s.store.UserRepository.GetByID(ctx, vt.UserID)
	if err != nil {
		return fmt.Errorf("%s: get user: %w", op, err)
	}

	if err := domain.DefaultPasswordPolicy.Validate(input.NewPassword, user.Username, user.Email); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	history, err := s.store.CredentialRepository.GetHistory(ctx, user.ID, domain.DefaultPasswordPolicy.HistoryCount)
	if err != nil {
		return fmt.Errorf("%s: get password history: %w", op, err)
	}

	for _, h := range history {
		if s.hasher.Verify(input.NewPassword, h.SecretHash) {
			return fmt.Errorf("%s: password reuse not allowed", op)
		}
	}

	passwordHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("%s: hash password: %w", op, err)
	}

	oldCred, err := s.store.CredentialRepository.GetCurrentByUserID(ctx, user.ID)
	version := 1
	if err == nil {
		oldCred.IsCurrent = false
		now := time.Now()
		oldCred.ExpiresAt = &now
		version = oldCred.Version + 1
		if err := s.store.CredentialRepository.Update(ctx, oldCred); err != nil {
			return fmt.Errorf("%s: retire old credential: %w", op, err)
		}
	}

	now := time.Now()
	newCred := &domain.UserCredential{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		CredentialType: domain.CredentialTypePassword,
		SecretHash:     passwordHash,
		Version:        version,
		IsCurrent:      true,
		CreatedAt:      now,
	}

	if err := s.store.CredentialRepository.Create(ctx, newCred); err != nil {
		return fmt.Errorf("%s: create credential: %w", op, err)
	}

	user.PasswordChangedAt = &now
	user.UpdatedAt = now
	user.RequirePasswordChange = false

	if err := s.store.UserRepository.Update(ctx, user); err != nil {
		return fmt.Errorf("%s: update user: %w", op, err)
	}

	if err := s.store.VerificationTokenRepository.MarkUsed(ctx, vt.ID); err != nil {
		return fmt.Errorf("%s: mark token used: %w", op, err)
	}

	return nil
}

type ChangePasswordInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

func (s *IdentityService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	const op = "IdentityService.ChangePassword"

	user, err := s.store.UserRepository.GetByID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("%s: get user: %w", op, err)
	}

	currentCred, err := s.store.CredentialRepository.GetCurrentByUserID(ctx, input.UserID)
	if err != nil {
		return fmt.Errorf("%s: get current credential: %w", op, err)
	}

	if !s.hasher.Verify(input.CurrentPassword, currentCred.SecretHash) {
		return fmt.Errorf("%s: current password invalid", op)
	}

	if err := domain.DefaultPasswordPolicy.Validate(input.NewPassword, user.Username, user.Email); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	history, err := s.store.CredentialRepository.GetHistory(ctx, user.ID, domain.DefaultPasswordPolicy.HistoryCount)
	if err != nil {
		return fmt.Errorf("%s: get password history: %w", op, err)
	}

	for _, h := range history {
		if s.hasher.Verify(input.NewPassword, h.SecretHash) {
			return fmt.Errorf("%s: password reuse not allowed", op)
		}
	}

	passwordHash, err := s.hasher.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("%s: hash password: %w", op, err)
	}

	now := time.Now()
	currentCred.IsCurrent = false
	currentCred.ExpiresAt = &now
	if err := s.store.CredentialRepository.Update(ctx, currentCred); err != nil {
		return fmt.Errorf("%s: retire old credential: %w", op, err)
	}

	newCred := &domain.UserCredential{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         user.ID,
		CredentialType: domain.CredentialTypePassword,
		SecretHash:     passwordHash,
		Version:        currentCred.Version + 1,
		IsCurrent:      true,
		CreatedAt:      now,
	}

	if err := s.store.CredentialRepository.Create(ctx, newCred); err != nil {
		return fmt.Errorf("%s: create credential: %w", op, err)
	}

	user.PasswordChangedAt = &now
	user.UpdatedAt = now

	if err := s.store.UserRepository.Update(ctx, user); err != nil {
		return fmt.Errorf("%s: update user: %w", op, err)
	}

	return nil
}
