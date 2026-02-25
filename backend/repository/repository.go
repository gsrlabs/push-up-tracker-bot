// Пакет repository обеспечивает взаимодействие с базой данных PostgreSQL
// и реализует слой доступа к данным для статистики отжиманий
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PushupRepository interface {
	Pool() *pgxpool.Pool
	EnsureUser(ctx context.Context, userID int64, username string) error
	AddPushups(ctx context.Context, userID int64, date time.Time, count int) error
	GetFullStat(ctx context.Context, userID int64, date time.Time) (*FullStatData, error)
	GetTodayStat(ctx context.Context, userID int64, date time.Time) (int, error)
	GetUsername(ctx context.Context, userID int64) (string, error)
	SetMaxReps(ctx context.Context, userID int64, count int) error
	SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error
	GetUserMaxReps(ctx context.Context, userID int64) (int, error)
	ResetDailyNorm(ctx context.Context, userID int64) error
	SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error
	GetDailyNorm(ctx context.Context, userID int64) (int, error)
	GetFirstWorkoutDate(ctx context.Context, userID int64) (time.Time, error)
	GetFirstNormCompleter(ctx context.Context, date time.Time) (int64, error)
	AddMaxRepsHistory(ctx context.Context, userID int64, maxReps int) error
	GetMaxRepsHistory(ctx context.Context, userID int64) ([]MaxRepsHistoryItem, error)
	GetMaxRepsRecord(ctx context.Context, userID int64) (MaxRepsHistoryItem, error)
}

// PushupRepository предоставляет методы для работы с данными отжиманий в БД
type pushupRepository struct {
	pool *pgxpool.Pool // Пул соединений с PostgreSQL
}

type LeaderboardItem struct {
	Rank     int // Будет добавляться в сервисе
	Username string
	Count    int
}

type FullStatData struct {
	TodayTotal       int
	TotalAllTime     int
	DailyNorm        int
	FirstWorkoutDate *time.Time
	Leaderboard      []LeaderboardItem
}

type MaxRepsHistoryItem struct {
	Date    time.Time
	MaxReps int
}

// NewPushupRepository создает новый экземпляр репозитория
func NewPushupRepository(pool *pgxpool.Pool) PushupRepository {
	return &pushupRepository{pool: pool}
}

func (r *pushupRepository) Pool() *pgxpool.Pool {
	return r.pool
}

// EnsureUser создает или обновляет пользователя
func (r *pushupRepository) EnsureUser(ctx context.Context, userID int64, username string) error {
	query := `
    INSERT INTO users (user_id, username)
    VALUES ($1, $2)
    ON CONFLICT (user_id) 
    DO UPDATE SET 
        username = EXCLUDED.username`

	_, err := r.pool.Exec(ctx, query, userID, username)
	return err
}

func (r *pushupRepository) AddPushups(ctx context.Context, userID int64, date time.Time, count int) error {
	query := `
	INSERT INTO pushups (user_id, date, count)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id, date)
	DO UPDATE SET 
		count = pushups.count + EXCLUDED.count`
	_, err := r.pool.Exec(ctx, query, userID, date, count)
	return err
}

func (r *pushupRepository) GetFullStat(
	ctx context.Context,
	userID int64,
	date time.Time,
) (*FullStatData, error) {

	query := `
WITH user_stats AS (
    SELECT
        COALESCE(SUM(count) FILTER (WHERE date = $2), 0) AS today_total,
        COALESCE(SUM(count), 0) AS total_all_time,
        MIN(date) AS first_date
    FROM pushups
    WHERE user_id = $1
),
leaderboard AS (
    SELECT json_agg(
        json_build_object(
            'username', username,
            'count', total_count
        )
        ORDER BY total_count DESC
    ) AS data
    FROM (
        SELECT u.username, SUM(p.count) AS total_count
        FROM pushups p
        JOIN users u ON u.user_id = p.user_id
        WHERE p.date = $2
        GROUP BY u.username
    ) t
)
SELECT
    us.today_total,
    us.total_all_time,
    u.daily_norm,
    us.first_date,
    COALESCE(lb.data, '[]'::json)
FROM users u
CROSS JOIN user_stats us
LEFT JOIN leaderboard lb ON true
WHERE u.user_id = $1;
	`

	var result FullStatData
	var leaderboardJSON []byte

	err := r.pool.QueryRow(ctx, query, userID, date).
		Scan(
			&result.TodayTotal,
			&result.TotalAllTime,
			&result.DailyNorm,
			&result.FirstWorkoutDate,
			&leaderboardJSON,
		)

	if err != nil {
		return nil, err
	}

	if len(leaderboardJSON) > 0 {
		if err := json.Unmarshal(leaderboardJSON, &result.Leaderboard); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// GetTodayStat возвращает суммарное количество отжиманий пользователя за указанную дату
func (r *pushupRepository) GetTodayStat(ctx context.Context, userID int64, date time.Time) (int, error) {
	query := `SELECT COALESCE(SUM(count), 0) FROM pushups WHERE user_id = $1 AND date = $2`
	var total int
	err := r.pool.QueryRow(ctx, query, userID, date.Truncate(24*time.Hour)).Scan(&total)
	return total, err
}

// GetUsername возвращает username пользователя
func (r *pushupRepository) GetUsername(ctx context.Context, userID int64) (string, error) {
	query := `SELECT username FROM users WHERE user_id = $1`
	var username string
	err := r.pool.QueryRow(ctx, query, userID).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("ошибка получения username: %w", err)
	}
	return username, nil
}

// SetMaxReps теперь обновляет и timestamp
func (r *pushupRepository) SetMaxReps(ctx context.Context, userID int64, count int) error {
	query := `UPDATE users SET max_reps = $1, last_updated_max_reps = CURRENT_TIMESTAMP WHERE user_id = $2`
	_, err := r.pool.Exec(ctx, query, count, userID)
	return err
}

// SetDateCompletionOfDailyNorm установка даты выполнения дневной нормы
func (r *pushupRepository) SetDateCompletionOfDailyNorm(ctx context.Context, userID int64) error {
	query := `UPDATE users SET last_updated = CURRENT_TIMESTAMP WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// GetLastMaxRepsUpdate возвращает дату последнего обновления max_reps
func (r *pushupRepository) GetLastMaxRepsUpdate(ctx context.Context, userID int64) (time.Time, error) {
	query := `SELECT last_updated_max_reps FROM users WHERE user_id = $1`
	var lastUpdate time.Time
	err := r.pool.QueryRow(ctx, query, userID).Scan(&lastUpdate)
	if err != nil {
		return time.Time{}, err
	}
	return lastUpdate, nil
}

func (r *pushupRepository) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	query := `SELECT max_reps FROM users WHERE user_id = $1`
	var maxReps int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&maxReps)
	return maxReps, err
}

// ResetMaxReps сбрасывает max_reps и daily_norm пользователя на значение по умолчанию
func (r *pushupRepository) ResetDailyNorm(ctx context.Context, userID int64) error {
	query := `UPDATE users SET daily_norm = 40 WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *pushupRepository) SetDailyNorm(ctx context.Context, userID int64, dailyNorm int) error {
	query := `UPDATE users SET daily_norm = $1 WHERE user_id = $2`
	_, err := r.pool.Exec(ctx, query, dailyNorm, userID)
	return err
}

// GetDailyNorm возвращает дневную норму пользователя
func (r *pushupRepository) GetDailyNorm(ctx context.Context, userID int64) (int, error) {
	query := `SELECT daily_norm FROM users WHERE user_id = $1`
	var dailyNorm int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&dailyNorm)
	return dailyNorm, err
}

// GetFirstWorkoutDate возвращает дату первой тренировки пользователя
func (r *pushupRepository) GetFirstWorkoutDate(ctx context.Context, userID int64) (time.Time, error) {
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

func (r *pushupRepository) GetFirstNormCompleter(ctx context.Context, date time.Time) (int64, error) {
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

// AddMaxRepsHistory добавляет запись об отжиманиях за подход в историю
func (r *pushupRepository) AddMaxRepsHistory(ctx context.Context, userID int64, maxReps int) error {
	query := `
    INSERT INTO max_reps_history (user_id, date, max_reps) 
    VALUES ($1, CURRENT_DATE, $2)
    ON CONFLICT (user_id, date) 
    DO UPDATE SET max_reps = $2`

	_, err := r.pool.Exec(ctx, query, userID, maxReps)
	return err
}

// GetMaxRepsHistory возвращает историю об отжиманий за подход пользователя
func (r *pushupRepository) GetMaxRepsHistory(ctx context.Context, userID int64) ([]MaxRepsHistoryItem, error) {
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

func (r *pushupRepository) GetMaxRepsRecord(ctx context.Context, userID int64) (MaxRepsHistoryItem, error) {
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
