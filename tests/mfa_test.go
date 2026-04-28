package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ihsansolusi/auth7/internal/domain"
	"github.com/ihsansolusi/auth7/internal/service/mfa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMFAConfigHasMethods(t *testing.T) {
	cfg := &domain.MFAConfig{
		ID:                    uuid.New(),
		UserID:                uuid.New(),
		IsTOTPEnabled:         true,
		IsEmailOTPEnabled:     true,
		IsBackupCodesEnabled:  true,
		TOTPSecretEncrypted:   []byte("encrypted"),
		BackupCodesHash:       []string{"hash1", "hash2"},
	}

	assert.True(t, cfg.HasTOTP())
	assert.True(t, cfg.HasEmailOTP())
	assert.True(t, cfg.HasBackupCodes())
	assert.True(t, cfg.IsFullyEnabled())

	cfg.IsTOTPEnabled = false
	assert.False(t, cfg.HasTOTP())
	assert.True(t, cfg.IsFullyEnabled())
}

func TestMFAConfigNotEnabled(t *testing.T) {
	cfg := &domain.MFAConfig{
		ID:                    uuid.New(),
		UserID:                uuid.New(),
		IsTOTPEnabled:         false,
		IsEmailOTPEnabled:     false,
		IsBackupCodesEnabled: false,
	}

	assert.False(t, cfg.HasTOTP())
	assert.False(t, cfg.HasEmailOTP())
	assert.False(t, cfg.HasBackupCodes())
	assert.False(t, cfg.IsFullyEnabled())
}

func TestEmailOTPCodeIsValid(t *testing.T) {
	code := &domain.EmailOTPCode{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Code:      "123456",
		Purpose:   domain.OTPPurposeLogin,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}

	assert.True(t, code.IsValid())
	assert.False(t, code.IsExpired())

	used := time.Now()
	code.UsedAt = &used
	assert.False(t, code.IsValid())

	code.UsedAt = nil
	code.ExpiresAt = time.Now().Add(-1 * time.Minute)
	assert.False(t, code.IsValid())
	assert.True(t, code.IsExpired())
}

type mockTOTPStore struct {
	configs map[uuid.UUID]*domain.MFAConfig
}

func (m *mockTOTPStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error) {
	if cfg, ok := m.configs[userID]; ok {
		return cfg, nil
	}
	return nil, nil
}

func (m *mockTOTPStore) Create(ctx context.Context, cfg *domain.MFAConfig) error {
	m.configs[cfg.UserID] = cfg
	return nil
}

func (m *mockTOTPStore) Update(ctx context.Context, cfg *domain.MFAConfig) error {
	m.configs[cfg.UserID] = cfg
	return nil
}

type mockTOTPRedis struct {
	data map[string]string
}

func (m *mockTOTPRedis) Get(ctx context.Context, key string) (string, error) {
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (m *mockTOTPRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.data[key] = "1"
	return nil
}

func (m *mockTOTPRedis) Del(ctx context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.data, k)
	}
	return nil
}

func TestTOTPEnroll(t *testing.T) {
	store := &mockTOTPStore{configs: make(map[uuid.UUID]*domain.MFAConfig)}
	redis := &mockTOTPRedis{data: make(map[string]string)}

	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}

	encryptor, err := mfa.NewEncryptor(encryptionKey)
	require.NoError(t, err)

	totpSvc := mfa.NewTOTPService(store, encryptor, redis)

	userID := uuid.New()
	output, err := totpSvc.Enroll(context.Background(), userID)
	require.NoError(t, err)
	assert.NotEmpty(t, output.Secret)
	assert.NotEmpty(t, output.QRCodeData)
	assert.Contains(t, output.QRCodeData, "otpauth://totp/")
}

func TestTOTPVerifyInvalidCode(t *testing.T) {
	store := &mockTOTPStore{configs: make(map[uuid.UUID]*domain.MFAConfig)}
	redis := &mockTOTPRedis{data: make(map[string]string)}

	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}

	encryptor, err := mfa.NewEncryptor(encryptionKey)
	require.NoError(t, err)

	totpSvc := mfa.NewTOTPService(store, encryptor, redis)

	userID := uuid.New()
	_, err = totpSvc.Enroll(context.Background(), userID)
	require.NoError(t, err)

	err = totpSvc.Verify(context.Background(), userID, "000000")
	assert.Error(t, err)
}

func TestBackupCodeService(t *testing.T) {
	store := &mockBackupStore{configs: make(map[uuid.UUID]*domain.MFAConfig)}

	backupSvc := mfa.NewBackupCodeService(store)

	userID := uuid.New()
	output, err := backupSvc.Generate(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, output.Codes, 10)

	for _, code := range output.Codes {
		assert.Len(t, code, 10)
	}

	remaining, err := backupSvc.Remaining(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, 10, remaining)

	err = backupSvc.Verify(context.Background(), userID, output.Codes[0])
	require.NoError(t, err)

	remaining, err = backupSvc.Remaining(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, 9, remaining)

	err = backupSvc.Verify(context.Background(), userID, "INVALID")
	assert.Error(t, err)
}

type mockBackupStore struct {
	configs map[uuid.UUID]*domain.MFAConfig
}

func (m *mockBackupStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.MFAConfig, error) {
	if cfg, ok := m.configs[userID]; ok {
		return cfg, nil
	}
	return nil, nil
}

func (m *mockBackupStore) Create(ctx context.Context, cfg *domain.MFAConfig) error {
	m.configs[cfg.UserID] = cfg
	return nil
}

func (m *mockBackupStore) Update(ctx context.Context, cfg *domain.MFAConfig) error {
	m.configs[cfg.UserID] = cfg
	return nil
}

func TestMFAMethodConstants(t *testing.T) {
	assert.Equal(t, domain.MFAMethod(""), domain.MFAMethodNone)
	assert.Equal(t, domain.MFAMethod("totp"), domain.MFAMethodTOTP)
	assert.Equal(t, domain.MFAMethod("email_otp"), domain.MFAMethodEmailOTP)
}

func TestOTPPurposeConstants(t *testing.T) {
	assert.Equal(t, "mfa_login", domain.OTPPurposeLogin)
	assert.Equal(t, "mfa_enroll", domain.OTPPurposeEnroll)
	assert.Equal(t, "change_email", domain.OTPPurposeChangeEmail)
}

func TestEncryptor(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := mfa.NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("my-secret-key")
	ciphertext, iv, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEmpty(t, iv)

	decrypted, err := encryptor.Decrypt(ciphertext, iv)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptorInvalidKey(t *testing.T) {
	_, err := mfa.NewEncryptor([]byte("short"))
	assert.Error(t, err)
}

func TestEncryptorDecryptMismatch(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := mfa.NewEncryptor(key)
	require.NoError(t, err)

	plaintext := []byte("my-secret-key")
	ciphertext, iv, err := encryptor.Encrypt(plaintext)
	require.NoError(t, err)

	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 1)
	}

	wrongEncryptor, err := mfa.NewEncryptor(wrongKey)
	require.NoError(t, err)

	_, err = wrongEncryptor.Decrypt(ciphertext, iv)
	assert.Error(t, err)
}

func TestTOTPGenerateCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)
	assert.Len(t, code, 6)

	valid := mfa.ValidateTOTPCode(secret, code)
	assert.True(t, valid)

	invalid := mfa.ValidateTOTPCode(secret, "000000")
	assert.False(t, invalid)
}