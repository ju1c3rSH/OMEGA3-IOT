package service

import (
	"OMEGA3-IOT/internal/repository"
	"context"
	"time"
)

type TokenBlacklistService struct {
	repo repository.TokenBlacklistRepository
}

func NewTokenBlacklistService(repo repository.TokenBlacklistRepository) *TokenBlacklistService {
	return &TokenBlacklistService{repo: repo}
}

func (s *TokenBlacklistService) BlacklistToken(ctx context.Context, jti string, remainingTTL time.Duration) error {
	return s.repo.BlacklistToken(ctx, jti, remainingTTL)
}

func (s *TokenBlacklistService) IsBlacklisted(ctx context.Context, jti string, userUUID string, issuedAt int64) (bool, error) {
	// Check individual token blacklist
	blacklisted, err := s.repo.IsTokenBlacklisted(ctx, jti)
	if err != nil {
		return false, err
	}
	if blacklisted {
		return true, nil
	}

	// Check user-level invalidation
	invalidationTime, err := s.repo.GetUserInvalidationTime(ctx, userUUID)
	if err != nil {
		return false, err
	}
	if invalidationTime > 0 && issuedAt < invalidationTime {
		return true, nil
	}

	return false, nil
}

func (s *TokenBlacklistService) InvalidateAllUserTokens(ctx context.Context, userUUID string) error {
	return s.repo.InvalidateAllUserTokens(ctx, userUUID)
}