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

// NewPushupRepository создает новый экземпляр репозитория
// Принимает:
// - pool: пул соединений с базой данных
// Возвращает:
// - *PushupRepository: инициализированный репозиторий
func NewPushupRepository(pool *pgxpool.Pool) *PushupRepository {

	
// logger := log.New(os.Stdout, "SQL: ", log.LstdFlags)
// pool.Config().ConnConfig.Tracer = &tracelog.TraceLog{
//     Logger:   tracelog.LoggerFunc(logger.Printf),
//     LogLevel: tracelog.LogLevelDebug,
// }
	return &PushupRepository{pool: pool}
}

// AddPushups добавляет указанное количество отжиманий для пользователя на указанную дату
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
	INSERT INTO pushups (user_id, date, count)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id, date) 
	DO UPDATE SET count = pushups.count + EXCLUDED.count`
	
	_, err := r.pool.Exec(ctx, query, userID, date.Truncate(24*time.Hour), count)
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



