package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type KeyManager struct {
	privateKey *rsa.PrivateKey
	kid       string
	algorithm string
	createdAt time.Time
}

func NewKeyManager(bits int) (*KeyManager, error) {
	if bits < 2048 {
		return nil, fmt.Errorf("key size must be at least 2048 bits")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	kid := fmt.Sprintf("auth7-%s", time.Now().Format("2006-01"))

	return &KeyManager{
		privateKey: privateKey,
		kid:        kid,
		algorithm:  "RS256",
		createdAt:  time.Now(),
	}, nil
}

func (km *KeyManager) Kid() string {
	return km.kid
}

func (km *KeyManager) Algorithm() string {
	return km.algorithm
}

func (km *KeyManager) PublicKey() *rsa.PublicKey {
	return &km.privateKey.PublicKey
}

func (km *KeyManager) PrivateKey() *rsa.PrivateKey {
	return km.privateKey
}

func (km *KeyManager) PublicKeyPEM() string {
	pubASN1, err := x509.MarshalPKIXPublicKey(&km.privateKey.PublicKey)
	if err != nil {
		return ""
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})

	return string(pubPEM)
}

func (km *KeyManager) JWKS() map[string]interface{} {
	n := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		km.privateKey.PublicKey.N.Bytes(),
	)
	e := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(
		big.NewInt(int64(km.privateKey.PublicKey.E)).Bytes(),
	)

	return map[string]interface{}{
		"kty": "RSA",
		"alg": km.algorithm,
		"use": "sig",
		"kid": km.kid,
		"n":   n,
		"e":   e,
	}
}

type RotatedKeyManager struct {
	keys     map[string]*KeyManager
	activeKid string
}

func NewRotatedKeyManager() *RotatedKeyManager {
	return &RotatedKeyManager{
		keys:     make(map[string]*KeyManager),
	}
}

func (rm *RotatedKeyManager) GenerateNewKey() (*KeyManager, error) {
	km, err := NewKeyManager(2048)
	if err != nil {
		return nil, err
	}

	rm.keys[km.kid] = km
	rm.activeKid = km.kid

	return km, nil
}

func (rm *RotatedKeyManager) GetKey(kid string) (*KeyManager, bool) {
	km, ok := rm.keys[kid]
	return km, ok
}

func (rm *RotatedKeyManager) ActiveKey() (*KeyManager, bool) {
	if rm.activeKid == "" {
		return nil, false
	}
	return rm.GetKey(rm.activeKid)
}

func (rm *RotatedKeyManager) AllKeys() []*KeyManager {
	keys := make([]*KeyManager, 0, len(rm.keys))
	for _, km := range rm.keys {
		keys = append(keys, km)
	}
	return keys
}

func (rm *RotatedKeyManager) JWKS() []map[string]interface{} {
	keys := make([]map[string]interface{}, 0, len(rm.keys))
	for _, km := range rm.keys {
		keys = append(keys, km.JWKS())
	}
	return keys
}
