package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type TodayCache struct {
	Mu      sync.RWMutex
	Items   map[int64]int
	changed bool

	filename string
	saveMu   sync.Mutex
}

const filename = "today_cache.json"

// NewTodayCache создает кэш и пытается загрузить данные из файла
func NewTodayCache() *TodayCache {
	c := &TodayCache{
		Items:    make(map[int64]int),
		filename: filename,
	}
	if err := c.Load(); err != nil {
		log.Printf("Не удалось загрузить кэш (%s): %v", filename, err)
	}
	go c.autoSaveLoop()
	return c
}

// Add добавляет указанное количество отжиманий для пользователя
func (c *TodayCache) Add(userID int64, count int) int {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	current := c.Items[userID]
	newTotal := current + count
	c.Items[userID] = newTotal
	c.changed = true
	return newTotal
}

// Get возвращает количество отжиманий пользователя за сегодня
func (c *TodayCache) Get(userID int64) int {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return c.Items[userID]
}

// Set устанавливает явное значение
func (c *TodayCache) Set(userID int64, total int) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Items[userID] = total
	c.changed = true
}

// ResetDaily сбрасывает кэш в полночь
func (c *TodayCache) ResetDaily() {
	for {
		now := time.Now()
		nextMidnight := now.Add(24 * time.Hour).Truncate(24 * time.Hour)
		time.Sleep(time.Until(nextMidnight))

		c.Mu.Lock()
		c.Items = make(map[int64]int)
		c.changed = true
		c.Mu.Unlock()

		_ = c.Save() // сразу сохраняем пустой кэш
		log.Printf("Cache reset at %v", time.Now())
	}
}

// Save сохраняет данные в файл
func (c *TodayCache) Save() error {
	c.saveMu.Lock()
	defer c.saveMu.Unlock()

	c.Mu.RLock()
	data, err := json.MarshalIndent(c.Items, "", "  ")
	c.Mu.RUnlock()
	if err != nil {
		return err
	}

	return os.WriteFile(c.filename, data, 0644)
}

// Load загружает данные из файла
func (c *TodayCache) Load() error {
	data, err := os.ReadFile(c.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // файла нет — просто пустой кэш
		}
		return err
	}

	var items map[int64]int
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	c.Mu.Lock()
	c.Items = items
	c.Mu.Unlock()

	return nil
}

// autoSaveLoop периодически сохраняет изменения
func (c *TodayCache) autoSaveLoop() {
	ticker := time.NewTicker(5 * time.Second) // каждые 5 секунд проверка
	defer ticker.Stop()

	for range ticker.C {
		c.Mu.Lock()
		changed := c.changed
		c.changed = false
		c.Mu.Unlock()

		if changed {
			if err := c.Save(); err != nil {
				log.Printf("Ошибка сохранения кэша: %v", err)
			}
		}
	}
}


//
// 🔹 Методы для отладки
//

// Size возвращает количество пользователей в кэше
func (c *TodayCache) Size() int {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return len(c.Items)
}

// Dump возвращает строку с содержимым кэша
func (c *TodayCache) Dump() string {
	c.Mu.RLock()
	defer c.Mu.RUnlock()

	if len(c.Items) == 0 {
		return "Кэш пуст 🚫"
	}

	out := "📊 Содержимое кэша:\n"
	for userID, count := range c.Items {
		out += fmt.Sprintf("👤 User %d → %d отжиманий\n", userID, count)
	}
	return out
}