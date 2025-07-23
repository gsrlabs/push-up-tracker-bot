// Пакет cache предоставляет кеширование дневной статистики отжиманий
// для уменьшения нагрузки на базу данных и быстрого доступа к данным
package cache

import (
	"log"
	"sync"
	"time"
)

// TodayCache представляет кеш дневной статистики отжиманий
// Потокобезопасен: использует sync.RWMutex для конкурентного доступа
type TodayCache struct {
	mu    sync.RWMutex     // Мьютекс для безопасного доступа из разных горутин
	items map[int64]int    // Хранилище данных: userID -> количество отжиманий
}

// NewTodayCache создает и возвращает новый экземпляр кеша
// Инициализирует внутреннее хранилище данных
func NewTodayCache() *TodayCache {
	return &TodayCache{
		items: make(map[int64]int),
	}
}

// Add добавляет указанное количество отжиманий для пользователя
// и возвращает новое суммарное значение за сегодня
//
// Параметры:
// - userID: идентификатор пользователя
// - count: количество отжиманий для добавления
//
// Возвращает:
// - int: новое общее количество отжиманий за сегодня
func (c *TodayCache) Add(userID int64, count int) int {
	c.mu.Lock()         // Блокировка на запись
	defer c.mu.Unlock() // Гарантированное освобождение блокировки

	current := c.items[userID]
	newTotal := current + count
	c.items[userID] = newTotal
	return newTotal
}

// Get возвращает количество отжиманий пользователя за сегодня
// Если данных нет - возвращает 0
//
// Параметры:
// - userID: идентификатор пользователя
//
// Возвращает:
// - int: количество отжиманий за сегодня
func (c *TodayCache) Get(userID int64) int {
	c.mu.RLock()         // Блокировка на чтение
	defer c.mu.RUnlock() // Гарантированное освобождение блокировки
	return c.items[userID]
}

// ResetDaily сбрасывает кеш ежедневно в полночь
// Должен запускаться в отдельной горутине при старте приложения
func (c *TodayCache) ResetDaily() {
	for {
		now := time.Now()
		// Вычисляем время следующей полночи
		nextMidnight := now.Add(24 * time.Hour).Truncate(24 * time.Hour)
		// Ожидаем до следующей полночи
		time.Sleep(time.Until(nextMidnight))
		
		// Сбрасываем кеш
		c.mu.Lock()
		c.items = make(map[int64]int) // Создаем новую чистую мапу
		c.mu.Unlock()
		log.Printf("Cache reset at %v", time.Now())
		
	}
}

// Set устанавливает явное значение количества отжиманий для пользователя
// Используется для синхронизации с данными из базы
//
// Параметры:
// - userID: идентификатор пользователя
// - total: общее количество отжиманий
func (c *TodayCache) Set(userID int64, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[userID] = total
}

// Метрики
// func (c *TodayCache) Size() int {
//     c.mu.RLock()
//     defer c.mu.RUnlock()
//     return len(c.items)
// }