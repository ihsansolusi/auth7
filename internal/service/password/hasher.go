package password

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	KeyLength   uint32
	SaltLength  uint32
}

func DefaultConfig() Config {
	return Config{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 4,
		KeyLength:   32,
		SaltLength:  16,
	}
}

type Hasher struct {
	config Config
}

func NewHasher(cfg Config) *Hasher {
	if cfg.Memory == 0 {
		cfg = DefaultConfig()
	}
	return &Hasher{config: cfg}
}

func (h *Hasher) Hash(password string) (string, error) {
	salt, err := generateSalt(h.config.SaltLength)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Iterations,
		h.config.Memory,
		h.config.Parallelism,
		h.config.KeyLength,
	)

	hashB64 := base64.RawStdEncoding.EncodeToString(hash)
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.config.Memory,
		h.config.Iterations,
		h.config.Parallelism,
		saltB64,
		hashB64,
	), nil
}

func (h *Hasher) Verify(password, hash string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}

	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}

	expected := argon2.IDKey(
		[]byte(password),
		salt,
		h.config.Iterations,
		h.config.Memory,
		h.config.Parallelism,
		h.config.KeyLength,
	)

	return subtle.ConstantTimeCompare(hashBytes, expected) == 1
}

func generateSalt(length uint32) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rng.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return salt, nil
}
