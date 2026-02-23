package service

import (
	"context"
	"fmt"

	"time"
	"trackerbot/cache"
	"trackerbot/repository"
)

type PushupService interface {
	EnsureUser(ctx context.Context, userID int64, username string) error
	AddPushups(ctx context.Context, userID int64, username string, count int) (*AddPushupsResult, error)
	SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error
	SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error
	GetDailyNorm(ctx context.Context, userID int64) (int, error)
	SetMaxReps(ctx context.Context, userID int64, username string, count int) error
	GetMaxRepsHistory(ctx context.Context, userID int64) ([]repository.MaxRepsHistoryItem, error)
	GetMaxRepsRecord(ctx context.Context, userID int64) (repository.MaxRepsHistoryItem, error)
	ResetDailyNorm(ctx context.Context, userID int64) error
	GetTodayStat(ctx context.Context, userID int64) (int, error)
	GetTotalStat(ctx context.Context, userID int64) (int, error)
	GetTodayLeaderboard(ctx context.Context) ([]repository.LeaderboardItem, error)
	GetUserMaxReps(ctx context.Context, userID int64) (int, error)
	GetFirstWorkoutDate(ctx context.Context, userID int64) (string, error)
	CheckNormCompletion(ctx context.Context) (bool, string)
	DebugCache() *cache.TodayCache
}

type AddPushupsResult struct {
	TotalToday int
	DailyNorm  int
}

type pushupService struct {
	repo     repository.PushupRepository
	cache    *cache.TodayCache
	location *time.Location
}

func NewPushupService(repo repository.PushupRepository, cache *cache.TodayCache, location *time.Location) PushupService {
	return &pushupService{
		repo:     repo,
		cache:    cache,
		location: location,
	}
}
func (s *pushupService) today() time.Time {
	now := time.Now().In(s.location)
	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		s.location,
	)
}

func (s *pushupService) EnsureUser(ctx context.Context, userID int64, username string) error {
	return s.repo.EnsureUser(ctx, userID, username)
}

func (s *pushupService) AddPushups(ctx context.Context, userID int64, username string, count int) (*AddPushupsResult, error) {

	today := s.today()

	dailyNorm, err := s.repo.GetDailyNorm(ctx, userID)
	if err != nil {
		return nil, err
	}

	if s.cache.Get(userID) == 0 {
		dbTotal, _ := s.repo.GetTodayStat(ctx, userID, today)
		s.cache.Set(userID, dbTotal)
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

func (s *pushupService) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	return s.repo.SetDailyNorm(ctx, userID, dailyNorm)
}

func (s *pushupService) SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error {
	return s.repo.SetDateCompletionOfDailyNorm(ctx, userID)
}

func (s *pushupService) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetDailyNorm(ctx, userID)
}

// SetMaxReps теперь также сохраняет в историю
func (s *pushupService) SetMaxReps(ctx context.Context, userID int64, username string, count int) error {
	// Сохраняем в основную таблицу пользователей
	if err := s.repo.SetMaxReps(ctx, userID, count); err != nil {
		return err
	}

	// Сохраняем в историю
	if err := s.repo.AddMaxRepsHistory(ctx, userID, count); err != nil {
		return fmt.Errorf("ошибка сохранения в историю: %w", err)
	}

	return nil
}

// GetMaxRepsHistory возвращает историю максимальных отжиманий
func (s *pushupService) GetMaxRepsHistory(ctx context.Context, userID int64) ([]repository.MaxRepsHistoryItem, error) {
	return s.repo.GetMaxRepsHistory(ctx, userID)
}

// GetMaxRepsHistory возвращает рекорд максимальных отжиманий
func (s *pushupService) GetMaxRepsRecord(ctx context.Context, userID int64) (repository.MaxRepsHistoryItem, error) {
	return s.repo.GetMaxRepsRecord(ctx, userID)
}

func (s *pushupService) ResetDailyNorm(ctx context.Context, userID int64) error {
	return s.repo.ResetDailyNorm(ctx, userID)
}

func (s *pushupService) GetTodayStat(ctx context.Context, userID int64) (int, error) {
	cached := s.cache.Get(userID)
	if cached >= 0 {
		return cached, nil
	}

	today := s.today()
	total, err := s.repo.GetTodayStat(ctx, userID, today)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения статистики: %w", err)
	}

	s.cache.Set(userID, total)

	return total, nil
}

func (s *pushupService) GetTotalStat(ctx context.Context, userID int64) (int, error) {
	total, err := s.repo.GetTotalStat(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения общей статистики: %w", err)
	}
	return total, nil
}

func (s *pushupService) GetTodayLeaderboard(ctx context.Context) ([]repository.LeaderboardItem, error) {
	items, err := s.repo.GetTodayLeaderboard(ctx)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Rank = i + 1
	}
	return items, nil
}

func (s *pushupService) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	return s.repo.GetUserMaxReps(ctx, userID)
}

func (s *pushupService) GetFirstWorkoutDate(ctx context.Context, userID int64) (string, error) {
	date, err := s.repo.GetFirstWorkoutDate(ctx, userID)
	if err != nil {
		return "", err
	}
	return date.Format("02.01.2006"), nil
}

// CheckNormCompletion проверяет, выполнил ли кто-то дневную норму
func (s *pushupService) CheckNormCompletion(ctx context.Context) (bool, string) {
	today := s.today()
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

func (s *pushupService) DebugCache() *cache.TodayCache {
	return s.cache
}
