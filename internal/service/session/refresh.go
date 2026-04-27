package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RefreshTokenData struct {
	TokenID   string `json:"token_id"`
	FamilyID  string `json:"family_id"`
	TokenHash string `json:"token_hash"`
	UserID    string `json:"user_id"`
	OrgID     string `json:"org_id"`
	ClientID  string `json:"client_id"`
	SessionID string `json:"session_id"`
	Scopes    []string `json:"scopes"`
	IssuedAt  int64  `json:"issued_at"`
	ExpiresAt int64  `json:"expires_at"`
	UsedAt    *int64 `json:"used_at,omitempty"`
	RevokedAt *int64 `json:"revoked_at,omitempty"`
}

type RefreshTokenStore struct {
	redis *redis.Client
}

func NewRefreshTokenStore(redis *redis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{redis: redis}
}

func (s *RefreshTokenStore) tokenKey(tokenHash string) string {
	return fmt.Sprintf("refresh:%s", tokenHash)
}

func (s *RefreshTokenStore) familyKey(familyID string) string {
	return fmt.Sprintf("refresh_family:%s", familyID)
}

func (s *RefreshTokenStore) Create(ctx context.Context, data *RefreshTokenData) error {
	const op = "RefreshTokenStore.Create"

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	ttl := time.Duration(data.ExpiresAt-time.Now().Unix()) * time.Second
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}

	pipe := s.redis.Pipeline()
	pipe.Set(ctx, s.tokenKey(data.TokenHash), jsonData, ttl)
	pipe.SAdd(ctx, s.familyKey(data.FamilyID), data.TokenHash)
	pipe.Expire(ctx, s.familyKey(data.FamilyID), ttl)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("%s: redis exec: %w", op, err)
	}

	return nil
}

func (s *RefreshTokenStore) GetByHash(ctx context.Context, tokenHash string) (*RefreshTokenData, error) {
	const op = "RefreshTokenStore.GetByHash"

	data, err := s.redis.Get(ctx, s.tokenKey(tokenHash)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: redis get: %w", op, err)
	}

	var token RefreshTokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("%s: unmarshal: %w", op, err)
	}

	return &token, nil
}

func (s *RefreshTokenStore) MarkUsed(ctx context.Context, tokenHash string) error {
	const op = "RefreshTokenStore.MarkUsed"

	token, err := s.GetByHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	if token == nil {
		return fmt.Errorf("token not found")
	}

	now := time.Now().Unix()
	token.UsedAt = &now

	jsonData, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	ttl := time.Duration(token.ExpiresAt-time.Now().Unix()) * time.Second

	return s.redis.Set(ctx, s.tokenKey(tokenHash), jsonData, ttl).Err()
}

func (s *RefreshTokenStore) Revoke(ctx context.Context, tokenHash string) error {
	const op = "RefreshTokenStore.Revoke"

	token, err := s.GetByHash(ctx, tokenHash)
	if err != nil {
		return err
	}

	if token == nil {
		return nil
	}

	now := time.Now().Unix()
	token.RevokedAt = &now

	jsonData, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	ttl := time.Duration(token.ExpiresAt-time.Now().Unix()) * time.Second
	if ttl <= 0 {
		return nil
	}

	return s.redis.Set(ctx, s.tokenKey(tokenHash), jsonData, ttl).Err()
}

func (s *RefreshTokenStore) RevokeFamily(ctx context.Context, familyID string) error {
	const op = "RefreshTokenStore.RevokeFamily"

	tokenHashes, err := s.redis.SMembers(ctx, s.familyKey(familyID)).Result()
	if err != nil {
		return fmt.Errorf("%s: smembers: %w", op, err)
	}

	now := time.Now().Unix()

	for _, hash := range tokenHashes {
		token, err := s.GetByHash(ctx, hash)
		if err != nil || token == nil {
			continue
		}

		if token.RevokedAt == nil {
			token.RevokedAt = &now
			jsonData, _ := json.Marshal(token)
			ttl := time.Duration(token.ExpiresAt-time.Now().Unix()) * time.Second
			if ttl > 0 {
				s.redis.Set(ctx, s.tokenKey(hash), jsonData, ttl)
			}
		}
	}

	return nil
}

func (s *RefreshTokenStore) IsReuse(ctx context.Context, tokenHash string) (bool, error) {
	const op = "RefreshTokenStore.IsReuse"

	token, err := s.GetByHash(ctx, tokenHash)
	if err != nil {
		return false, err
	}

	if token == nil {
		return false, nil
	}

	return token.UsedAt != nil, nil
}

func (s *RefreshTokenStore) GetFamilyTokens(ctx context.Context, familyID string) ([]*RefreshTokenData, error) {
	const op = "RefreshTokenStore.GetFamilyTokens"

	tokenHashes, err := s.redis.SMembers(ctx, s.familyKey(familyID)).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: smembers: %w", op, err)
	}

	var tokens []*RefreshTokenData
	for _, hash := range tokenHashes {
		token, err := s.GetByHash(ctx, hash)
		if err != nil || token == nil {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens, nil
}

type BlacklistStore struct {
	redis *redis.Client
}

func NewBlacklistStore(redis *redis.Client) *BlacklistStore {
	return &BlacklistStore{redis: redis}
}

func (s *BlacklistStore) blacklistKey(jti string) string {
	return fmt.Sprintf("blacklist:%s", jti)
}

func (s *BlacklistStore) Add(ctx context.Context, jti string, ttl time.Duration) error {
	const op = "BlacklistStore.Add"

	if ttl <= 0 {
		return nil
	}

	return s.redis.Set(ctx, s.blacklistKey(jti), "1", ttl).Err()
}

func (s *BlacklistStore) Contains(ctx context.Context, jti string) (bool, error) {
	const op = "BlacklistStore.Contains"

	exists, err := s.redis.Exists(ctx, s.blacklistKey(jti)).Result()
	if err != nil {
		return false, fmt.Errorf("%s: redis exists: %w", op, err)
	}

	return exists > 0, nil
}

func (s *BlacklistStore) AddSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	const op = "BlacklistStore.AddSession"

	sessionKey := fmt.Sprintf("blacklist_session:%s", sessionID)
	return s.redis.Set(ctx, sessionKey, "1", ttl).Err()
}

func (s *BlacklistStore) IsSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	const op = "BlacklistStore.IsSessionRevoked"

	sessionKey := fmt.Sprintf("blacklist_session:%s", sessionID)
	exists, err := s.redis.Exists(ctx, sessionKey).Result()
	if err != nil {
		return false, fmt.Errorf("%s: redis exists: %w", op, err)
	}

	return exists > 0, nil
}
