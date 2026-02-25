package service

import (
	"bytes"
	"context"
	"fmt"

	"time"
	"trackerbot/cache"
	"trackerbot/presenter"
	"trackerbot/repository"
)

type PushupService interface {
	EnsureUser(ctx context.Context, userID int64, username string) error
	AddPushups(ctx context.Context, userID int64, count int) (*presenter.AddPushupsViewModel, error)
	SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error
	SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error
	GetDailyNorm(ctx context.Context, userID int64) (int, error)
	UpdateMaxReps(ctx context.Context, userID int64, count int) (*presenter.MaxRepsViewModel, error)
	GetMaxRepsHistory(ctx context.Context, userID int64) ([]repository.MaxRepsHistoryItem, error)
	GetMaxRepsRecord(ctx context.Context, userID int64) (repository.MaxRepsHistoryItem, error)
	ResetDailyNorm(ctx context.Context, userID int64) error
	GetFullStat(ctx context.Context, userID int64) (*presenter.FullStatViewModel, error)
	GetUserMaxReps(ctx context.Context, userID int64) (int, error)
	GetFirstWorkoutDate(ctx context.Context, userID int64) (string, error)
	CheckNormCompletion(ctx context.Context) (bool, string)
	BuildSchedule(ctx context.Context, userID int64, history []repository.MaxRepsHistoryItem) (bytes.Buffer, error)
	DebugCache() *cache.TodayCache
}

type pushupService struct {
	repo     repository.PushupRepository
	cache    *cache.TodayCache
	location *time.Location
}

type AddPushupsResult struct {
	TotalToday        int
	DailyNorm         int
	NormJustCompleted bool
	HasLeader         bool
	LeaderName        string
	ResponseText      string
}

type UpdateMaxRepsResult struct {
	DailyNorm    int
	ResponseText string
}

type FullStatResult struct {
	ResponseText string
}

type FullStatViewModel struct {
	TodayTotal       int
	TotalAllTime     int
	DailyNorm        int
	FirstWorkoutDate *time.Time
	Leaderboard      []repository.LeaderboardItem
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

func (s *pushupService) AddPushups(
	ctx context.Context,
	userID int64,
	count int,
) (*presenter.AddPushupsViewModel, error) {

	today := s.today()

	// --- Получаем дневную норму ---
	dailyNorm, err := s.repo.GetDailyNorm(ctx, userID)
	if err != nil {
		return nil, err
	}

	// --- Инициализируем кэш, если пустой ---
	if s.cache.Get(userID) == 0 {
		dbTotal, _ := s.repo.GetTodayStat(ctx, userID, today)
		s.cache.Set(userID, dbTotal)
	}

	// --- Добавляем отжимания ---
	if err := s.repo.AddPushups(ctx, userID, today, count); err != nil {
		return nil, fmt.Errorf("ошибка сохранения в БД: %w", err)
	}

	totalToday := s.cache.Add(userID, count)

	// --- Проверяем выполнение дневной нормы ---
	hasCompleted, firstCompleter := s.CheckNormCompletion(ctx)
	normJustCompleted := totalToday >= dailyNorm
	if normJustCompleted {
		_ = s.repo.SetDateCompletionOfDailyNorm(ctx, userID)
	}

	// --- Формируем ViewModel ---
	vm := &presenter.AddPushupsViewModel{
		AddedCount: count,
		Total:      totalToday,
		DailyNorm:  dailyNorm,
		Completed:  normJustCompleted,
		HasLeader:  hasCompleted,
		Leader:     firstCompleter,
	}

	return vm, nil
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

func (s *pushupService) UpdateMaxReps(
	ctx context.Context,
	userID int64,
	count int,
) (*presenter.MaxRepsViewModel, error) {

	// 1. Сохраняем max reps и историю
	if err := s.repo.SetMaxReps(ctx, userID, count); err != nil {
		return nil, err
	}
	if err := s.repo.AddMaxRepsHistory(ctx, userID, count); err != nil {
		return nil, fmt.Errorf("ошибка сохранения в историю: %w", err)
	}

	// 2. Рассчитываем дневную норму
	dailyNorm := CalculateDailyNorm(count)
	if err := s.repo.SetDailyNorm(ctx, userID, dailyNorm); err != nil {
		return nil, err
	}

	// 3. Получаем историю и рекорд
	history, err := s.repo.GetMaxRepsHistory(ctx, userID)
	if err != nil {
		return nil, err
	}
	record, err := s.repo.GetMaxRepsRecord(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 4. Формируем ViewModel
	vm := &presenter.MaxRepsViewModel{
		Count:      count,
		DailyNorm:  dailyNorm,
		History:    history,
		Record:     &record,
		Rank:       GetUserRank(count),
		RepsToNext: GetRepsToNextRank(count),
	}

	return vm, nil
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

func (s *pushupService) GetFullStat(
	ctx context.Context,
	userID int64,
) (*presenter.FullStatViewModel, error) {

	today := s.today()

	data, err := s.repo.GetFullStat(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	vm := &presenter.FullStatViewModel{
		TodayTotal:       data.TodayTotal,
		TotalAllTime:     data.TotalAllTime,
		DailyNorm:        data.DailyNorm,
		FirstWorkoutDate: data.FirstWorkoutDate,
	}

	for _, item := range data.Leaderboard {
		vm.Leaderboard = append(vm.Leaderboard, presenter.LeaderboardItem{
			Username: item.Username,
			Count:    item.Count,
		})
	}

	return vm, nil
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

func (s *pushupService) BuildSchedule(ctx context.Context,
	userID int64,
	history []repository.MaxRepsHistoryItem,
) (bytes.Buffer, error) {

	imageBytes, err := SendSchedule(userID, history)
	if err != nil {
		return bytes.Buffer{}, nil
	}

	return imageBytes, nil
}

func (s *pushupService) DebugCache() *cache.TodayCache {
	return s.cache
}
