// Пакет repository обеспечивает взаимодействие с базой данных PostgreSQL
// и реализует слой доступа к данным для статистики отжиманий
package repository

import (
	"context"
	"fmt"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PushupRepository предоставляет методы для работы с данными отжиманий в БД
type PushupRepository struct {
	pool *pgxpool.Pool // Пул соединений с PostgreSQL
}

type LeaderboardItem struct {
	Rank     int // Будет добавляться в сервисе
	Username string
	Count    int
}

type MaxRepsHistoryItem struct {
	Date    time.Time
	MaxReps int
}

// NewPushupRepository создает новый экземпляр репозитория
func NewPushupRepository(pool *pgxpool.Pool) *PushupRepository {
	return &PushupRepository{pool: pool}
}

func (r *PushupRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// EnsureUser создает или обновляет пользователя
func (r *PushupRepository) EnsureUser(ctx context.Context, userID int64, username string) error {
	query := `
    INSERT INTO users (user_id, username)
    VALUES ($1, $2)
    ON CONFLICT (user_id) 
    DO UPDATE SET 
        username = EXCLUDED.username`

	_, err := r.pool.Exec(ctx, query, userID, username)
	return err
}

func (r *PushupRepository) AddPushups(ctx context.Context, userID int64, date time.Time, count int) error {
	query := `INSERT INTO pushups (user_id, date, count) VALUES ($1, $2, $3)`
	_, err := r.pool.Exec(ctx, query, userID, date, count)
	return err
}

// GetTodayStat возвращает суммарное количество отжиманий пользователя за указанную дату
func (r *PushupRepository) GetTodayStat(ctx context.Context, userID int64, date time.Time) (int, error) {
	query := `SELECT COALESCE(SUM(count), 0) FROM pushups WHERE user_id = $1 AND date = $2`
	var total int
	err := r.pool.QueryRow(ctx, query, userID, date.Truncate(24*time.Hour)).Scan(&total)
	return total, err
}

// GetTotalStat возвращает суммарное количество отжиманий пользователя за все время
func (r *PushupRepository) GetTotalStat(ctx context.Context, userID int64) (int, error) {
	query := `SELECT COALESCE(SUM(count), 0) FROM pushups WHERE user_id = $1`
	var total int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

// New methods for leaderboards
func (r *PushupRepository) GetTodayLeaderboard(ctx context.Context) ([]LeaderboardItem, error) {
	query := `
    SELECT u.username, SUM(p.count) AS count
    FROM pushups p
    JOIN users u ON p.user_id = u.user_id
    WHERE p.date = CURRENT_DATE
    GROUP BY u.username
    ORDER BY count DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []LeaderboardItem
	for rows.Next() {
		var item LeaderboardItem
		if err := rows.Scan(&item.Username, &item.Count); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// GetUsername возвращает username пользователя
func (r *PushupRepository) GetUsername(ctx context.Context, userID int64) (string, error) {
	query := `SELECT username FROM users WHERE user_id = $1`
	var username string
	err := r.pool.QueryRow(ctx, query, userID).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("ошибка получения username: %w", err)
	}
	return username, nil
}

func (r *PushupRepository) SetMaxReps(ctx context.Context, userID int64, count int) error {
	query := `UPDATE users SET max_reps = $1 WHERE user_id = $2`
	_, err := r.pool.Exec(ctx, query, count, userID)
	return err
}

func (r *PushupRepository) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	query := `SELECT max_reps FROM users WHERE user_id = $1`
	var maxReps int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&maxReps)
	return maxReps, err
}

// ResetMaxReps сбрасывает max_reps и daily_norm пользователя на значение по умолчанию
func (r *PushupRepository) ResetMaxReps(ctx context.Context, userID int64) error {
	query := `UPDATE users SET max_reps = 0, daily_norm = 40 WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *PushupRepository) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	query := `UPDATE users SET daily_norm = $1 WHERE user_id = $2`
	_, err := r.pool.Exec(ctx, query, dailyNorm, userID)
	return err
}

// GetDailyNorm возвращает дневную норму пользователя
func (r *PushupRepository) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	query := `SELECT daily_norm FROM users WHERE user_id = $1`
	var dailyNorm int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&dailyNorm)
	return dailyNorm, err
}

// Добавляем методы для управления напоминаниями
func (r *PushupRepository) DisableNotifications(ctx context.Context, userID int64) error {
	query := `UPDATE users SET notifications_enabled = FALSE WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *PushupRepository) EnableNotifications(ctx context.Context, userID int64) error {
	query := `UPDATE users SET notifications_enabled = TRUE WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *PushupRepository) GetNotificationsStatus(ctx context.Context, userID int64) (bool, error) {
	query := `SELECT notifications_enabled FROM users WHERE user_id = $1`
	var enabled bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&enabled)
	return enabled, err
}

// GetFirstWorkoutDate возвращает дату первой тренировки пользователя
func (r *PushupRepository) GetFirstWorkoutDate(ctx context.Context, userID int64) (time.Time, error) {
	query := `
        SELECT COALESCE(MIN(date), '0001-01-01'::DATE) 
        FROM pushups 
        WHERE user_id = $1 
        AND count > 0
    `

	var firstDate time.Time
	err := r.pool.QueryRow(ctx, query, userID).Scan(&firstDate)

	if err != nil {
		return time.Time{}, fmt.Errorf("ошибка получения даты первой тренировки: %w", err)
	}

	return firstDate, nil
}

func (r *PushupRepository) GetFirstNormCompleter(ctx context.Context, date time.Time) (int64, error) {
	query := `
        SELECT user_id 
        FROM pushups 
        WHERE date = $1 
        GROUP BY user_id 
        HAVING SUM(count) >= (SELECT daily_norm FROM users WHERE user_id = pushups.user_id)
        ORDER BY MIN(record_id) 
        LIMIT 1
    `

	var userID int64
	err := r.pool.QueryRow(ctx, query, date).Scan(&userID)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// repository/repository.go

// AddMaxRepsHistory добавляет запись об отжиманиях за подход в историю
func (r *PushupRepository) AddMaxRepsHistory(ctx context.Context, userID int64, maxReps int) error {
	query := `
    INSERT INTO max_reps_history (user_id, date, max_reps) 
    VALUES ($1, CURRENT_DATE, $2)
    ON CONFLICT (user_id, date) 
    DO UPDATE SET max_reps = $2`

	_, err := r.pool.Exec(ctx, query, userID, maxReps)
	return err
}

// GetMaxRepsHistory возвращает историю об отжиманий за подход пользователя
func (r *PushupRepository) GetMaxRepsHistory(ctx context.Context, userID int64) ([]MaxRepsHistoryItem, error) {
	query := `
    SELECT date, max_reps 
    FROM max_reps_history 
    WHERE user_id = $1 
    ORDER BY date DESC 
    LIMIT 60` // Ограничиваем 60 последними записями

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []MaxRepsHistoryItem
	for rows.Next() {
		var item MaxRepsHistoryItem
		if err := rows.Scan(&item.Date, &item.MaxReps); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, nil
}

func (r *PushupRepository) GetMaxRepsRecord(ctx context.Context, userID int64) (MaxRepsHistoryItem, error) {
	query := `
    SELECT date, max_reps 
	FROM max_reps_history 
	WHERE user_id = $1 
	ORDER BY max_reps DESC, date DESC 
	LIMIT 1`

	var maxRepsRecord MaxRepsHistoryItem

	err := r.pool.QueryRow(ctx, query, userID).Scan(&maxRepsRecord.Date, &maxRepsRecord.MaxReps)
	if err != nil {
		return MaxRepsHistoryItem{}, err
	}

	return maxRepsRecord, nil
}
