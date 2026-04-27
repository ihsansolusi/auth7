package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/password"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserValidation(t *testing.T) {
	user := &domain.User{
		ID:        uuid.New(),
		OrgID:     uuid.New(),
		Username:  "john.doe",
		Email:     "john@example.com",
		FullName:  "John Doe",
		Status:    domain.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := user.Validate()
	assert.NoError(t, err)
}

func TestUserCanLogin(t *testing.T) {
	user := &domain.User{
		Status: domain.UserStatusActive,
	}

	assert.True(t, user.CanLogin())

	user.Status = domain.UserStatusLocked
	assert.False(t, user.CanLogin())

	user.Status = domain.UserStatusActive
	future := time.Now().Add(1 * time.Hour)
	user.LockedUntil = &future
	assert.False(t, user.CanLogin())
}

func TestPasswordHashing(t *testing.T) {
	hasher := password.NewHasher(password.DefaultConfig())

	hash, err := hasher.Hash("SecurePassword123!")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, hasher.Verify("SecurePassword123!", hash))
	assert.False(t, hasher.Verify("WrongPassword123!", hash))
}

func TestPasswordPolicy(t *testing.T) {
	policy := domain.DefaultPasswordPolicy

	err := policy.Validate("Short1!", "john", "john@example.com")
	assert.Error(t, err)

	err = policy.Validate("alllowercase1!", "john", "john@example.com")
	assert.Error(t, err)

	err = policy.Validate("NoNumber!", "john", "john@example.com")
	assert.Error(t, err)

	err = policy.Validate("ValidPass1", "johndoe", "johndoe@example.com")
	assert.NoError(t, err)
}

func TestPasswordPolicyRejectsPasswordWithUsername(t *testing.T) {
	policy := domain.DefaultPasswordPolicy

	err := policy.Validate("Passwordjohn1", "john", "john@example.com")
	assert.Error(t, err)
}

func TestPasswordPolicyWithDifferentUsers(t *testing.T) {
	policy := domain.DefaultPasswordPolicy

	err := policy.Validate("MyPassword123", "alice", "alice@example.com")
	assert.NoError(t, err)

	err = policy.Validate("MyPassword123", "bob", "bob@example.com")
	assert.NoError(t, err)
}

func TestVerificationToken(t *testing.T) {
	token := &domain.VerificationToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "test-token",
		TokenType: domain.TokenTypeEmailVerification,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	assert.True(t, token.IsValid())

	used := time.Now()
	token.UsedAt = &used
	assert.False(t, token.IsValid())
}

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"JOHN@EXAMPLE.COM", "john@example.com"},
		{"  john@example.com  ", "john@example.com"},
		{"John@Example.COM", "john@example.com"},
	}

	for _, tt := range tests {
		result := domain.NormalizeEmail(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestCredentialTypes(t *testing.T) {
	assert.Equal(t, "password", domain.CredentialTypePassword)
	assert.Equal(t, "api_key", domain.CredentialTypeAPIKey)
}

func TestTokenTypes(t *testing.T) {
	assert.Equal(t, "email_verification", domain.TokenTypeEmailVerification)
	assert.Equal(t, "password_recovery", domain.TokenTypePasswordRecovery)
}

func TestUserStatus(t *testing.T) {
	assert.Equal(t, domain.UserStatus("created"), domain.UserStatusCreated)
	assert.Equal(t, domain.UserStatus("pending_verification"), domain.UserStatusPendingVerification)
	assert.Equal(t, domain.UserStatus("active"), domain.UserStatusActive)
	assert.Equal(t, domain.UserStatus("locked"), domain.UserStatusLocked)
}

func TestMFAMethod(t *testing.T) {
	assert.Equal(t, domain.MFAMethod("totp"), domain.MFAMethodTOTP)
	assert.Equal(t, domain.MFAMethod("email_otp"), domain.MFAMethodEmailOTP)
}
