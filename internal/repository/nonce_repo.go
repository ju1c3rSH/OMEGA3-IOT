package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NonceRepository 定义 Nonce 存储接口
type NonceRepository interface {
	// StoreNonce 存储 Nonce（带 TTL）
	StoreNonce(ctx context.Context, username string, nonce string, ttl time.Duration) error
	// GetAndDeleteNonce 原子性获取并删除 Nonce（一次性使用）
	GetAndDeleteNonce(ctx context.Context, username string) (string, error)
}

// nonceRepo Nonce 存储实现
type nonceRepo struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration
}

// NonceRepoConfig Nonce 仓储配置
type NonceRepoConfig struct {
	KeyPrefix  string        // Redis key 前缀，默认 "auth:nonce:"
	DefaultTTL time.Duration // 默认 TTL，默认 60 秒
}

// DefaultNonceRepoConfig 返回默认配置
func DefaultNonceRepoConfig() NonceRepoConfig {
	return NonceRepoConfig{
		KeyPrefix:  "auth:nonce:",
		DefaultTTL: 60 * time.Second,
	}
}

// NewNonceRepository 创建 Nonce 仓储实例
func NewNonceRepository(client *redis.Client, cfg NonceRepoConfig) NonceRepository {
	return &nonceRepo{
		client:     client,
		keyPrefix:  cfg.KeyPrefix,
		defaultTTL: cfg.DefaultTTL,
	}
}

func (r *nonceRepo) nonceKey(username string) string {
	return fmt.Sprintf("%s%s", r.keyPrefix, username)
}

func (r *nonceRepo) StoreNonce(ctx context.Context, username string, nonce string, ttl time.Duration) error {
	key := r.nonceKey(username)
	if ttl == 0 {
		ttl = r.defaultTTL
	}
	return r.client.Set(ctx, key, nonce, ttl).Err()
}

func (r *nonceRepo) GetAndDeleteNonce(ctx context.Context, username string) (string, error) {
	key := r.nonceKey(username)

	// 使用 Lua 脚本实现原子性 GET + DELETE
	script := redis.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if val then
			redis.call("DEL", KEYS[1])
		end
		return val
	`)

	result, err := script.Run(ctx, r.client, []string{key}).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("nonce not found or already used")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	nonce, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("invalid nonce format in redis")
	}

	return nonce, nil
}
