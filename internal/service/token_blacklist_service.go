package service

import (
	"OMEGA3-IOT/internal/repository"
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Timeout for Redis blacklist checks; prevents hanging on Redis failure.
// Per https://oneuptime.com/blog/post/2026-03-31-redis-context-timeouts-go/view
// and circuit-breaker pattern https://abhishekshr.hashnode.dev/my-backend-crashed-every-time-redis-went-down-three-patterns-fixed-that
const (
	blacklistCheckTimeout        = 200 * time.Millisecond
	negativeCacheTTL             = 5 * time.Second
	negativeCacheCleanupInterval = 30 * time.Second
)

type negativeCacheEntry struct {
	expiresAt time.Time
}

type TokenBlacklistService struct {
	repo          repository.TokenBlacklistRepository
	negativeCache sync.Map // map[string]negativeCacheEntry, key=jti:userUUID:issuedAt, caches negative (not blacklisted) for 5s
}

func NewTokenBlacklistService(repo repository.TokenBlacklistRepository) *TokenBlacklistService {
	s := &TokenBlacklistService{repo: repo}
	go s.cleanupLoop()
	return s
}

// cleanupLoop periodically evicts expired negative-cache entries.
// Prevents unbounded growth when many unique JTIs are seen once and never reused.
// Pattern from https://dev.to/young_gao/building-a-high-performance-cache-layer-in-go-2ejd
// and https://reintech.io/blog/implementing-caching-in-go-using-sync-map-package
func (s *TokenBlacklistService) cleanupLoop() {
	ticker := time.NewTicker(negativeCacheCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.negativeCache.Range(func(k, v any) bool {
			e := v.(negativeCacheEntry)
			if now.After(e.expiresAt) {
				s.negativeCache.Delete(k)
			}
			return true
		})
	}
}

func (s *TokenBlacklistService) BlacklistToken(ctx context.Context, jti string, remainingTTL time.Duration) error {
	err := s.repo.BlacklistToken(ctx, jti, remainingTTL)
	if err == nil {
		// Evict any negative cache entries for this jti so revocation is visible immediately
		s.negativeCache.Range(func(k, _ any) bool {
			if ks, ok := k.(string); ok && strings.HasPrefix(ks, jti+":") {
				s.negativeCache.Delete(k)
			}
			return true
		})
	}
	return err
}

func (s *TokenBlacklistService) IsBlacklisted(ctx context.Context, jti string, userUUID string, issuedAt int64) (bool, error) {
	// Negative cache fast-path: avoid Redis entirely on 90% hits (not blacklisted).
	// Caches 5s, bounds staleness vs Redis load tradeoff.
	// See https://dev.to/young_gao/building-a-high-performance-cache-layer-in-go-2ejd (sync.Map + TTL)
	cacheKey := jti + ":" + userUUID + ":" + strconv.FormatInt(issuedAt, 10)
	if v, ok := s.negativeCache.Load(cacheKey); ok {
		e := v.(negativeCacheEntry)
		if time.Now().Before(e.expiresAt) {
			return false, nil
		}
		s.negativeCache.Delete(cacheKey)
	}

	// Per-call timeout to avoid hanging on Redis failure; fail-open on timeout.
	// See https://oneuptime.com/blog/post/2026-03-31-redis-context-timeouts-go/view
	timeoutCtx, cancel := context.WithTimeout(ctx, blacklistCheckTimeout)
	defer cancel()

	// Single RTT via pipeline (EXISTS + GET) instead of sequential 2 RTT.
	// See https://redis.io/docs/latest/develop/clients/go/transpipe/
	// and https://redis.uptrace.dev/guide/go-redis-pipelines.html
	blacklisted, err := s.repo.IsBlacklisted(timeoutCtx, jti, userUUID, issuedAt)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			log.Printf("[TokenBlacklist] Redis timeout fail-open (200ms) jti=%s user=%s err=%v", jti, userUUID, err)
			return false, nil
		}
		return false, err
	}
	if !blacklisted {
		s.negativeCache.Store(cacheKey, negativeCacheEntry{expiresAt: time.Now().Add(negativeCacheTTL)})
	}
	return blacklisted, nil
}

func (s *TokenBlacklistService) InvalidateAllUserTokens(ctx context.Context, userUUID string) error {
	err := s.repo.InvalidateAllUserTokens(ctx, userUUID)
	if err == nil {
		// Evict negative entries for this user so invalidation is visible immediately
		s.negativeCache.Range(func(k, _ any) bool {
			if ks, ok := k.(string); ok && strings.Contains(ks, ":"+userUUID+":") {
				s.negativeCache.Delete(k)
			}
			return true
		})
	}
	return err
}
