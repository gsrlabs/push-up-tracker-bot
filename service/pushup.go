package service

import (
	"context"
	"fmt"
	"time"
	"trackerbot/cache"
	"trackerbot/repository"
)

type AddPushupsResult struct {
	TotalToday int
	DailyNorm  int
}

type PushupService struct {
	repo  repository.PushupRepository
	cache *cache.TodayCache
}

func NewPushupService(repo repository.PushupRepository, cache *cache.TodayCache) *PushupService {
	return &PushupService{repo: repo, cache: cache}
}

func (s *PushupService) AddPushups(ctx context.Context, userID int64, username string, count int) (*AddPushupsResult, error) {
	if err := s.repo.EnsureUser(ctx, userID, username); err != nil {
		return nil, err
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	
	if err := s.repo.AddPushups(ctx, userID, today, count); err != nil {
		return nil, fmt.Errorf("ошибка сохранения в БД: %w", err)
	}
	totalToday := s.cache.Add(userID, count)

	maxReps, err := s.repo.GetUserMaxReps(ctx, userID)
	if err != nil {
		return nil, err
	}
	dailyNorm := CalculateDailyNorm(maxReps)

	return &AddPushupsResult{
		TotalToday: totalToday,
		DailyNorm:  dailyNorm,
	}, nil
}

func (s *PushupService) GetTodayStat(ctx context.Context, userID int64) (int, error) {
	if cached := s.cache.Get(userID); cached > 0 {
		return cached, nil
	}
	
	today := time.Now().UTC().Truncate(24 * time.Hour)
	total, err := s.repo.GetTodayStat(ctx, userID, today)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения статистики: %w", err)
	}
	
	s.cache.Set(userID, total)
	
	return total, nil
}

func (s *PushupService) GetTotalStat(ctx context.Context, userID int64) (int, error) {
	total, err := s.repo.GetTotalStat(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения общей статистики: %w", err)
	}
	return total, nil
}

func (s *PushupService) GetTodayLeaderboard(ctx context.Context) ([]repository.LeaderboardItem, error) {
	items, err := s.repo.GetTodayLeaderboard(ctx)
	if err != nil {
		return nil, err
	}
	
	for i := range items {
		items[i].Rank = i + 1
	}
	return items, nil
}

func (s *PushupService) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetUserMaxReps(ctx, userID)
}