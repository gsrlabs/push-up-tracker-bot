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
	return &PushupService{
		repo:  repo,
		cache: cache,
	}
}

func (s *PushupService) EnsureUser(ctx context.Context, userID int64, username string) error {
	return s.repo.EnsureUser(ctx, userID, username)
}

func (s *PushupService) AddPushups(ctx context.Context, userID int64, username string, count int) (*AddPushupsResult, error) {

	today := time.Now().UTC().Truncate(24 * time.Hour)

	dailyNorm, err := s.repo.GetDailyNorm(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.AddPushups(ctx, userID, today, count); err != nil {
		return nil, fmt.Errorf("ошибка сохранения в БД: %w", err)
	}

	totalToday := s.cache.Add(userID, count)

	return &AddPushupsResult{
		TotalToday: totalToday,
		DailyNorm:  dailyNorm,
	}, nil
}

func (s *PushupService) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	return s.repo.SetDailyNorm(ctx, userID, dailyNorm)
}

func (s *PushupService) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetDailyNorm(ctx, userID)
}

func (s *PushupService) SetMaxReps(ctx context.Context, userID int64, username string, count int) error {
	return s.repo.SetMaxReps(ctx, userID, count)
}


func (s *PushupService) ResetMaxReps(ctx context.Context, userID int64) error {
	return s.repo.ResetMaxReps(ctx, userID)
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

func (s *PushupService) GetFirstWorkoutDate(ctx context.Context, userID int64) (string, error) {
	date, err := s.repo.GetFirstWorkoutDate(ctx, userID)
	if err != nil {
		return "", err
	}
	return date.Format("02.01.2006"), nil
}

// CheckNormCompletion проверяет, выполнил ли кто-то дневную норму
func (s *PushupService) CheckNormCompletion(ctx context.Context, dailyNorm int) (bool, string) {
    today := time.Now().UTC().Truncate(24 * time.Hour)
    userID, err := s.repo.GetFirstNormCompleter(ctx, today)
    
    if err != nil || userID == 0 {
        return false, ""
    }
    
    username, err := s.repo.GetUsername(ctx, userID)
    if err != nil {
        username = fmt.Sprintf("User%d", userID)
    }
    
    return true, username
}

// Добавляем методы для управления напоминаниями в сервисе
func (s *PushupService) DisableNotifications(ctx context.Context, userID int64) error {
	return s.repo.DisableNotifications(ctx, userID)
}

func (s *PushupService) EnableNotifications(ctx context.Context, userID int64) error {
	return s.repo.EnableNotifications(ctx, userID)
}

func (s *PushupService) GetNotificationsStatus(ctx context.Context, userID int64) (bool, error) {
	return s.repo.GetNotificationsStatus(ctx, userID)
}

func (s *PushupService) DebugCache() *cache.TodayCache {
	return s.cache
}
