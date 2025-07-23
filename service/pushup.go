// Пакет service содержит бизнес-логику приложения для работы с отжиманиями
// и объединяет работу репозитория (база данных) и кеша
package service

import (
	"context"
	"fmt"
	"time"
	"trackerbot/cache"
	"trackerbot/repository"
)

// PushupService предоставляет методы для работы с данными отжиманий,
// комбинируя доступ к базе данных и кеширование
type PushupService struct {
	repo  repository.PushupRepository // Репозиторий для работы с БД
	cache *cache.TodayCache           // Кеш дневной статистики
}

// NewPushupService создает новый экземпляр сервиса
// Принимает:
// - repo: репозиторий для работы с базой данных
// - cache: кеш дневной статистики
// Возвращает:
// - *PushupService: инициализированный сервис
func NewPushupService(repo repository.PushupRepository, cache *cache.TodayCache) *PushupService {
	return &PushupService{repo: repo, cache: cache}
}

// AddPushups добавляет указанное количество отжиманий для текущего пользователя
// и возвращает текстовый отчет о выполнении дневной нормы
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
// - count: количество отжиманий для добавления
//
// Возвращает:
// - string: текстовый отчет о выполнении
// - error: ошибка операции или nil
func (s *PushupService) AddPushups(ctx context.Context, userID int64, count int) (string, error) {
	// Получаем текущую дату (без времени)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	
	// 1. Сохраняем в базу данных
	if err := s.repo.AddPushups(ctx, userID, today, count); err != nil {
		return "", fmt.Errorf("ошибка сохранения в БД: %w", err)
	}

	// 2. Обновляем кеш и получаем актуальное суммарное значение
	totalToday := s.cache.Add(userID, count)

	// 3. Формируем ответ пользователю
	response := fmt.Sprintf("Добавлено: %d отжиманий\nВаш прогресс: %d/100", count, totalToday)
	
	// Проверяем выполнение дневной нормы
	if totalToday >= 100 {
		response += "\nВы выполнили дневную норму!"
	}

	return response, nil
}

// GetTodayStat возвращает количество отжиманий пользователя за сегодня
// Сначала проверяет кеш, если данных нет - запрашивает из базы
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
//
// Возвращает:
// - int: количество отжиманий за сегодня
// - error: ошибка операции или nil
func (s *PushupService) GetTodayStat(ctx context.Context, userID int64) (int, error) {
	// 1. Проверяем кеш
	if cached := s.cache.Get(userID); cached > 0 {
		return cached, nil
	}
	
	// 2. Если в кеше нет данных, запрашиваем из БД
	today := time.Now().UTC().Truncate(24 * time.Hour)
	total, err := s.repo.GetTodayStat(ctx, userID, today)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения статистики: %w", err)
	}
	
	// 3. Обновляем кеш
	s.cache.Set(userID, total)
	
	return total, nil
}

// GetTotalStat возвращает общее количество отжиманий пользователя за все время
// Всегда запрашивает данные из базы (без кеширования)
//
// Параметры:
// - ctx: контекст выполнения
// - userID: идентификатор пользователя
//
// Возвращает:
// - int: общее количество отжиманий
// - error: ошибка операции или nil
func (s *PushupService) GetTotalStat(ctx context.Context, userID int64) (int, error) {
	total, err := s.repo.GetTotalStat(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения общей статистики: %w", err)
	}
	return total, nil
}