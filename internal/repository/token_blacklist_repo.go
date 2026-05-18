package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlacklistRepository interface {
	BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
	InvalidateAllUserTokens(ctx context.Context, userUUID string) error
	GetUserInvalidationTime(ctx context.Context, userUUID string) (int64, error)
}

type tokenBlacklistRepo struct {
	client *redis.Client
}

func NewTokenBlacklistRepository(client *redis.Client) TokenBlacklistRepository {
	return &tokenBlacklistRepo{client: client}
}

func (r *tokenBlacklistRepo) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:token:%s", jti)
	return r.client.Set(ctx, key, "1", ttl).Err()
}

func (r *tokenBlacklistRepo) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:token:%s", jti)
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (r *tokenBlacklistRepo) InvalidateAllUserTokens(ctx context.Context, userUUID string) error {
	key := fmt.Sprintf("blacklist:user:%s", userUUID)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return r.client.Set(ctx, key, ts, 7*24*time.Hour).Err()
}

func (r *tokenBlacklistRepo) GetUserInvalidationTime(ctx context.Context, userUUID string) (int64, error) {
	key := fmt.Sprintf("blacklist:user:%s", userUUID)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}
