package mfa

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	const op = "mfa.NewEncryptor"
	if len(key) != 32 {
		return nil, fmt.Errorf("%s: key must be 32 bytes", op)
	}
	return &Encryptor{key: key}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) (ciphertext, iv []byte, err error) {
	const op = "Encryptor.Encrypt"

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: new cipher: %w", op, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: new gcm: %w", op, err)
	}

	iv = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, fmt.Errorf("%s: random iv: %w", op, err)
	}

	ciphertext = gcm.Seal(nil, iv, plaintext, nil)
	return ciphertext, iv, nil
}

func (e *Encryptor) Decrypt(ciphertext, iv []byte) ([]byte, error) {
	const op = "Encryptor.Decrypt"

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%s: new cipher: %w", op, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%s: new gcm: %w", op, err)
	}

	if len(iv) != gcm.NonceSize() {
		return nil, fmt.Errorf("%s: invalid iv size", op)
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: decrypt: %w", op, err)
	}

	return plaintext, nil
}