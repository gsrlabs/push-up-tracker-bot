package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TodayCache struct {
	Mu      sync.RWMutex
	Items   map[int64]int
	changed bool

	filename string
	saveMu   sync.Mutex

	location *time.Location
}

// NewTodayCache создает кэш и запускает фоновые процессы
func NewTodayCache(ctx context.Context, location *time.Location) *TodayCache {
	c := &TodayCache{
		Items:    make(map[int64]int),
		filename: getCacheFilePath(),
		location: location,
	}

	// Создаем директорию cache если её нет
	if err := os.MkdirAll(filepath.Dir(c.filename), 0755); err != nil {
		log.Printf("Не удалось создать директорию для кэша: %v", err)
	}

	// Загружаем существующий кэш
	if err := c.Load(); err != nil {
		log.Printf("Не удалось загрузить кэш (%s): %v", c.filename, err)
	}

	// Запускаем фоновый автосейв
	go c.autoSaveLoop(ctx)

	// Запускаем цикл сброса кэша
	go c.resetDailyLoop(ctx)

	return c
}

func getCacheFilePath() string {
	// Получаем текущую рабочую директорию (должна быть корень проекта)
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Не удалось получить рабочую директорию: %v", err)
		// Fallback - используем относительный путь от корня проекта
		return "cache/today_cache.json"
	}
	
	return filepath.Join(wd, "cache", "today_cache.json")
}

// Add добавляет указанное количество отжиманий
func (c *TodayCache) Add(userID int64, count int) int {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	current := c.Items[userID]
	newTotal := current + count
	c.Items[userID] = newTotal
	c.changed = true
	return newTotal
}

// Get возвращает текущее значение
func (c *TodayCache) Get(userID int64) int {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return c.Items[userID]
}

// Set устанавливает значение
func (c *TodayCache) Set(userID int64, total int) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.Items[userID] = total
	c.changed = true
}

//
// 🔥 Фоновые циклы
//

func (c *TodayCache) autoSaveLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("autoSaveLoop stopping...")
			_ = c.Save()
			return
		case <-ticker.C:
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
}

func (c *TodayCache) resetDailyLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Инициализируем lastDay текущим днем, чтобы не сбрасывать кэш сразу
	lastDay := time.Now().In(c.location).YearDay()

	for {
		select {
		case <-ctx.Done():
			log.Println("resetDailyLoop stopping...")
			return
		case <-ticker.C:
			now := time.Now().In(c.location)
			currentDay := now.YearDay()

			if currentDay != lastDay {
				c.Mu.Lock()
				c.Items = make(map[int64]int)
				c.changed = true
				c.Mu.Unlock()

				if err := c.Save(); err != nil {
					log.Printf("Ошибка сохранения кэша при сбросе дня: %v", err)
				}

				log.Printf("Cache reset at %v", now)
				lastDay = currentDay
			}
		}
	}
}

//
// 💾 File operations
//

func (c *TodayCache) Save() error {
	c.saveMu.Lock()
	defer c.saveMu.Unlock()

	c.Mu.RLock()
	data, err := json.MarshalIndent(c.Items, "", "  ")
	c.Mu.RUnlock()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(c.filename), 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию: %w", err)
	}

	tmp := c.filename + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmp, c.filename); err != nil {
		return err
	}

	return nil
}

func (c *TodayCache) Load() error {
	data, err := os.ReadFile(c.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
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

//
// 🔹 Debug
//

func (c *TodayCache) Size() int {
	c.Mu.RLock()
	defer c.Mu.RUnlock()
	return len(c.Items)
}

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