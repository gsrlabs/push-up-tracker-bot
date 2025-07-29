// Пакет repository обеспечивает взаимодействие с базой данных PostgreSQL
// и реализует слой доступа к данным для статистики отжиманий
package repository

import (
	"context"
	"time"
	
	"github.com/jackc/pgx/v5/pgxpool"
	
)

// PushupRepository предоставляет методы для работы с данными отжиманий в БД
type PushupRepository struct {
	pool *pgxpool.Pool // Пул соединений с PostgreSQL
}

type LeaderboardItem struct {
    Rank     int    // Будет добавляться в сервисе
    Username string
    Count    int
}

// NewPushupRepository создает новый экземпляр репозитория
// Принимает:
// - pool: пул соединений с базой данных
// Возвращает:
// - *PushupRepository: инициализированный репозиторий
func NewPushupRepository(pool *pgxpool.Pool) *PushupRepository {

	return &PushupRepository{pool: pool}
}


// EnsureUser создает или обновляет пользователя
func (r *PushupRepository) EnsureUser(ctx context.Context, userID int64, username string) error {
    query := `
    INSERT INTO users (user_id, username)
    VALUES ($1, $2)
    ON CONFLICT (user_id) 
    DO UPDATE SET username = EXCLUDED.username`
    
    _, err := r.pool.Exec(ctx, query, userID, username)
    return err
}

// AddPushups добавляет указанное количество отжиманий для пользователя на указанную дату с учетом max_reps
// Если запись для пользователя и даты уже существует - увеличивает существующее значение
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - date: дата (обрезается до дня)
// - count: количество отжиманий для добавления
//
// Возвращает:
// - error: ошибка операции или nil
func (r *PushupRepository) AddPushups(ctx context.Context, userID int64, date time.Time, count int) error {
    query := `
    WITH new_pushups AS (
        INSERT INTO pushups (user_id, date, count)
        VALUES ($1, $2, $3)
        RETURNING user_id, count
    )
    UPDATE users u
    SET 
        max_reps = GREATEST(u.max_reps, np.count),
        last_updated = CURRENT_TIMESTAMP
    FROM new_pushups np
    WHERE u.user_id = np.user_id`
    
    _, err := r.pool.Exec(ctx, query, userID, date, count)
    return err
}

// GetTodayStat возвращает суммарное количество отжиманий пользователя за указанную дату
// Если данных нет - возвращает 0
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - date: дата для статистики
//
// Возвращает:
// - int: количество отжиманий
// - error: ошибка операции или nil
func (r *PushupRepository) GetTodayStat(ctx context.Context, userID int64, date time.Time) (int, error) {
	query := `SELECT COALESCE(SUM(count), 0) FROM pushups WHERE user_id = $1 AND date = $2`
	var total int
	err := r.pool.QueryRow(ctx, query, userID, date.Truncate(24*time.Hour)).Scan(&total)
	return total, err
}

// GetTotalStat возвращает суммарное количество отжиманий пользователя за все время
// Если данных нет - возвращает 0
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
//
// Возвращает:
// - int: общее количество отжиманий
// - error: ошибка операции или nil
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

func (r *PushupRepository) GetUserMaxReps(ctx context.Context, userID int64) (int, error) {
	query := `SELECT max_reps FROM users WHERE user_id = $1`
	var maxReps int
	err := r.pool.QueryRow(ctx, query, userID).Scan(&maxReps)
	return maxReps, err
}