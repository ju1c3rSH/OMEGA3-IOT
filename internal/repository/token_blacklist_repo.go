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
	// IsBlacklisted checks both per-token blacklist (EXISTS) and per-user
	// invalidation (GET) in a single Redis RTT via Pipeline.
	// See https://redis.io/docs/latest/develop/clients/go/transpipe/
	// and https://redis.uptrace.dev/guide/go-redis-pipelines.html
	IsBlacklisted(ctx context.Context, jti string, userUUID string, issuedAt int64) (bool, error)
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

// IsBlacklisted batches EXISTS blacklist:token:{jti} + GET blacklist:user:{uuid}
// into a single Redis pipeline (1 RTT instead of 2). On pipeline Exec failure the
// error is returned for caller to decide fail-open vs fail-closed.
// Uses Pipelined callback per https://redis.io/docs/latest/develop/clients/go/transpipe/
// Pitfall: per-command results are valid only after Exec; errors are exposed via
// Result() and must handle redis.Nil for missing user key.
func (r *tokenBlacklistRepo) IsBlacklisted(ctx context.Context, jti string, userUUID string, issuedAt int64) (bool, error) {
	tokenKey := fmt.Sprintf("blacklist:token:%s", jti)
	userKey := fmt.Sprintf("blacklist:user:%s", userUUID)

	var existsCmd *redis.IntCmd
	var getCmd *redis.StringCmd

	_, err := r.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		existsCmd = pipe.Exists(ctx, tokenKey)
		getCmd = pipe.Get(ctx, userKey)
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("pipeline exec failed: %w", err)
	}

	// Token blacklist check (EXISTS)
	if existsCmd != nil {
		exists, err := existsCmd.Result()
		if err != nil {
			return false, fmt.Errorf("pipeline Exists failed: %w", err)
		}
		if exists > 0 {
			return true, nil
		}
	}

	// User-level invalidation check (GET)
	if getCmd != nil {
		val, err := getCmd.Result()
		if err == redis.Nil {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("pipeline Get failed: %w", err)
		}
		invTime, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return false, fmt.Errorf("invalid invalidation time %q: %w", val, err)
		}
		if invTime > 0 && issuedAt < invTime {
			return true, nil
		}
	}

	return false, nil
}
