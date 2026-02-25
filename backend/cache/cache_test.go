package cache

import (
	"context"

	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTodayCache_AddGetSet(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC

	c := NewTodayCache(ctx, loc)

	userID := int64(1)

	// Изначально кэш пустой
	assert.Equal(t, 0, c.Get(userID))
	assert.Equal(t, 0, c.Size())

	// Добавляем значение
	total := c.Add(userID, 5)
	assert.Equal(t, 5, total)
	assert.Equal(t, 5, c.Get(userID))
	assert.Equal(t, 1, c.Size())

	// Добавляем еще
	total = c.Add(userID, 10)
	assert.Equal(t, 15, total)
	assert.Equal(t, 15, c.Get(userID))

	// Set напрямую
	c.Set(userID, 42)
	assert.Equal(t, 42, c.Get(userID))
}

func TestTodayCache_Dump(t *testing.T) {
	ctx := context.Background()
	loc := time.UTC
	c := NewTodayCache(ctx, loc)

	assert.Equal(t, "Кэш пуст 🚫", c.Dump())

	c.Set(1, 10)
	c.Set(2, 20)

	dump := c.Dump()
	assert.Contains(t, dump, "User 1 → 10 отжиманий")
	assert.Contains(t, dump, "User 2 → 20 отжиманий")
}

func TestTodayCache_SaveLoad(t *testing.T) {

	loc := time.UTC

	// Создаем временную директорию для кэша
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "cache.json")

	c := &TodayCache{
		Items:    make(map[int64]int),
		filename: filename,
		location: loc,
	}

	c.Set(1, 100)
	c.Set(2, 200)

	// Сохраняем
	err := c.Save()
	assert.NoError(t, err)

	// Создаем новый кэш и загружаем
	c2 := &TodayCache{
		Items:    make(map[int64]int),
		filename: filename,
		location: loc,
	}

	err = c2.Load()
	assert.NoError(t, err)

	assert.Equal(t, 100, c2.Get(1))
	assert.Equal(t, 200, c2.Get(2))
	assert.Equal(t, 2, c2.Size())
}

func TestTodayCache_AutoSaveLoopStopsOnContextCancel(t *testing.T) {
	loc := time.UTC
	ctx, cancel := context.WithCancel(context.Background())
	c := NewTodayCache(ctx, loc)

	c.Set(1, 1)

	// Даем немного времени, чтобы autoSaveLoop сработал хотя бы один раз
	time.Sleep(100 * time.Millisecond)

	cancel()
	// После отмены контекста autoSaveLoop должен завершиться
	time.Sleep(50 * time.Millisecond)
}

func TestTodayCache_ResetDailyLoop(t *testing.T) {
	loc := time.UTC
	ctx, cancel := context.WithCancel(context.Background())
	c := NewTodayCache(ctx, loc)

	c.Set(1, 100)
	assert.Equal(t, 100, c.Get(1))

	// Симулируем сброс вручную, без таймера
	c.Mu.Lock()
	c.Items = make(map[int64]int)
	c.Mu.Unlock()

	assert.Equal(t, 0, c.Get(1))

	cancel()
}
