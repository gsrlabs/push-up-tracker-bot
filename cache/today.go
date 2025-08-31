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
	Mu    sync.RWMutex     // Мьютекс для безопасного доступа из разных горутин
	Items map[int64]int    // Хранилище данных: userID -> количество отжиманий
}

// NewTodayCache создает и возвращает новый экземпляр кеша
// Инициализирует внутреннее хранилище данных
func NewTodayCache() *TodayCache {
	return &TodayCache{
		Items: make(map[int64]int),
	}
}

// Add добавляет указанное количество отжиманий для пользователя
func (c *TodayCache) Add(userID int64, count int) int {
	c.Mu.Lock()         // Блокировка на запись
	defer c.Mu.Unlock() // Гарантированное освобождение блокировки

	current := c.Items[userID]
	newTotal := current + count
	c.Items[userID] = newTotal
	return newTotal
}

// Get возвращает количество отжиманий пользователя за сегодня
func (c *TodayCache) Get(userID int64) int {
	c.Mu.RLock()         // Блокировка на чтение
	defer c.Mu.RUnlock() // Гарантированное освобождение блокировки
	return c.Items[userID]
}

// ResetDaily сбрасывает кеш ежедневно в полночь
func (c *TodayCache) ResetDaily() {
	for {
		now := time.Now()
		// Вычисляем время следующей полночи
		nextMidnight := now.Add(24 * time.Hour).Truncate(24 * time.Hour)
		// Ожидаем до следующей полночи
		time.Sleep(time.Until(nextMidnight))
		
		// Сбрасываем кеш
		c.Mu.Lock()
		c.Items = make(map[int64]int) // Создаем новую чистую мапу
		c.Mu.Unlock()
		log.Printf("Cache reset at %v", time.Now())
		
	}
}

// Set устанавливает явное значение количества отжиманий для пользователя
func (c *TodayCache) Set(userID int64, total int) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Items[userID] = total
}

// Метрики
//  func (c *TodayCache) Size() int {
//      c.mu.RLock()
//      defer c.mu.RUnlock()
//      return len(c.items)
// }