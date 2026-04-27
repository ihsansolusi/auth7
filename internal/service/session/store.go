package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionData struct {
	ID                string   `json:"id"`
	UserID            string   `json:"user_id"`
	OrgID             string   `json:"org_id"`
	ActiveBranchID    string   `json:"active_branch_id,omitempty"`
	AssignedBranchIDs []string `json:"assigned_branch_ids,omitempty"`
	Roles             []string `json:"roles,omitempty"`
	Permissions       []string `json:"permissions,omitempty"`
	IPAddress         string   `json:"ip_address"`
	UserAgent         string   `json:"user_agent"`
	DeviceInfo        string   `json:"device_info"`
	CreatedAt         int64    `json:"created_at"`
	ExpiresAt         int64    `json:"expires_at"`
	LastUsedAt        int64    `json:"last_used_at"`
	MFAVerified       bool     `json:"mfa_verified"`
}

type Store struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewStore(redis *redis.Client, ttl time.Duration) *Store {
	return &Store{
		redis: redis,
		ttl:   ttl,
	}
}

func (s *Store) sessionKey(sessionID string) string {
	return fmt.Sprintf("session:%s", sessionID)
}

func (s *Store) Create(ctx context.Context, data *SessionData) error {
	const op = "session.Store.Create"

	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	now := time.Now().Unix()
	data.CreatedAt = now
	data.LastUsedAt = now
	data.ExpiresAt = now + int64(s.ttl.Seconds())

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	err = s.redis.Set(ctx, s.sessionKey(data.ID), jsonData, s.ttl).Err()
	if err != nil {
		return fmt.Errorf("%s: redis set: %w", op, err)
	}

	return nil
}

func (s *Store) Get(ctx context.Context, sessionID string) (*SessionData, error) {
	const op = "session.Store.Get"

	data, err := s.redis.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: redis get: %w", op, err)
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("%s: unmarshal: %w", op, err)
	}

	return &session, nil
}

func (s *Store) Update(ctx context.Context, sessionID string, update func(*SessionData) error) error {
	const op = "session.Store.Update"

	data, err := s.redis.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("%s: session not found", op)
		}
		return fmt.Errorf("%s: redis get: %w", op, err)
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("%s: unmarshal: %w", op, err)
	}

	if err := update(&session); err != nil {
		return fmt.Errorf("%s: update fn: %w", op, err)
	}

	session.LastUsedAt = time.Now().Unix()

	jsonData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	ttl := time.Duration(session.ExpiresAt-time.Now().Unix()) * time.Second
	if ttl <= 0 {
		ttl = s.ttl
	}

	if err := s.redis.Set(ctx, s.sessionKey(sessionID), jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("%s: redis set: %w", op, err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, sessionID string) error {
	const op = "session.Store.Delete"

	err := s.redis.Del(ctx, s.sessionKey(sessionID)).Err()
	if err != nil {
		return fmt.Errorf("%s: redis del: %w", op, err)
	}

	return nil
}

func (s *Store) DeleteByUser(ctx context.Context, userID string) error {
	const op = "session.Store.DeleteByUser"

	pattern := "session:*"
	iter := s.redis.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		data, err := s.redis.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}

		var session SessionData
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		if session.UserID == userID {
			s.redis.Del(ctx, iter.Val())
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("%s: scan: %w", op, err)
	}

	return nil
}

func (s *Store) CheckIPBinding(ctx context.Context, sessionID, currentIP string) (bool, error) {
	const op = "session.Store.CheckIPBinding"

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}

	if session == nil {
		return false, nil
	}

	return session.IPAddress == currentIP, nil
}

func (s *Store) RevokeIPMismatch(ctx context.Context, sessionID, oldIP, newIP string) error {
	const op = "session.Store.RevokeIPMismatch"

	err := s.Delete(ctx, sessionID)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateLastUsed(ctx context.Context, sessionID string) error {
	return s.Update(ctx, sessionID, func(s *SessionData) error {
		s.LastUsedAt = time.Now().Unix()
		return nil
	})
}

func (s *Store) ExtendTTL(ctx context.Context, sessionID string, extension time.Duration) error {
	const op = "session.Store.ExtendTTL"

	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.ExpiresAt = time.Now().Add(extension).Unix()

	jsonData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("%s: marshal: %w", op, err)
	}

	return s.redis.Set(ctx, s.sessionKey(sessionID), jsonData, extension).Err()
}
