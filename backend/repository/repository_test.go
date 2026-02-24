package repository

import (
	"context"
	"log"

	"os"
	"testing"
	"time"

	"trackerbot/config"
	"trackerbot/db"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func getTestConfig(t *testing.T) *config.Config {
	envPaths := []string{
		"../../../.env",
		"../../.env",
		"../.env",
		".env",
	}

	for _, p := range envPaths {
		if err := godotenv.Load(p); err == nil {
			log.Printf("INFO: loaded env from %s", p)
			break
		}
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		panic("DB_PASSWORD is not set for tests")
	}

	configPaths := []string{
		"../../../config/config.yml",
		"../../config/config.yml",
		"../config/config.yml",
		"config/config.yml",
	}

	var cfg *config.Config
	var err error

	for _, p := range configPaths {
		cfg, err = config.Load(p)
		if err == nil {
			log.Printf("INFO: loaded config from %s", p)
			break
		}
	}

	if err != nil {
		panic("failed to load config.yml for tests")
	}

	cfg.Database.Password = dbPass

	// Если в структуре Config есть блок Test, используем его, иначе задаем дефолты
	if cfg.Test.DBHost != "" {
		cfg.Database.Host = cfg.Test.DBHost
	} else {
		cfg.Database.Host = "localhost"
	}

	migrationPath := cfg.Test.MigrationsPath

	if migrationPath == "" {
		migrationPaths := []string{
			"../../../backend/migrations",
			"../../backend/migrations",
			"../backend/migrations",
			"../../migrations",
			"../migrations",
			"./migrations",
		}

		for _, p := range migrationPaths {
			if _, err := os.Stat(p); err == nil {
				migrationPath = p
				log.Printf("INFO: using migrations from %s", p)
				break
			}
		}
	}

	if migrationPath == "" {
		panic("migrations directory not found")
	}

	cfg.Migrations.Path = migrationPath

	return cfg
}

func setupRepo(t *testing.T) PushupRepository {
	cfg := getTestConfig(t)
	ctx := context.Background()

	database, err := db.Connect(ctx, cfg)
	assert.NoError(t, err)
	return NewPushupRepository(database.GetPool())
}

func cleanUpUser(ctx context.Context, r PushupRepository, userID int64) {
	pool := r.Pool()
	_, _ = pool.Exec(ctx, "DELETE FROM max_reps_history WHERE user_id=$1", userID)
	_, _ = pool.Exec(ctx, "DELETE FROM pushups WHERE user_id=$1", userID)
	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE user_id=$1", userID)
}

func TestPushupRepository_CRUD(t *testing.T) {
	ctx := context.Background()
	repo := setupRepo(t)

	userID := int64(99999)
	username := "testuser"

	// Очистка на всякий случай перед тестом
	cleanUpUser(ctx, repo, userID)
	defer cleanUpUser(ctx, repo, userID) // Очистка после теста

	// 1️⃣ EnsureUser
	err := repo.EnsureUser(ctx, userID, username)
	assert.NoError(t, err)

	// 2️⃣ GetUsername
	uName, err := repo.GetUsername(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, username, uName)

	// 3️⃣ AddPushups
	today := time.Now().Truncate(24 * time.Hour)
	err = repo.AddPushups(ctx, userID, today, 10)
	assert.NoError(t, err)

	// 4️⃣ GetTodayStat
	todayTotal, err := repo.GetTodayStat(ctx, userID, today)
	assert.NoError(t, err)
	assert.Equal(t, 10, todayTotal)

	// 5️⃣ SetMaxReps
	err = repo.SetMaxReps(ctx, userID, 50)
	assert.NoError(t, err)

	// 6️⃣ GetUserMaxReps
	maxReps, err := repo.GetUserMaxReps(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 50, maxReps)

	// 7️⃣ ResetDailyNorm
	err = repo.ResetDailyNorm(ctx, userID)
	assert.NoError(t, err)
	dailyNorm, _ := repo.GetDailyNorm(ctx, userID)
	assert.Equal(t, 40, dailyNorm)

	// 8️⃣ SetDailyNorm
	err = repo.SetDailyNorm(ctx, userID, 60)
	assert.NoError(t, err)
	dailyNorm, _ = repo.GetDailyNorm(ctx, userID)
	assert.Equal(t, 60, dailyNorm)

	// 9️⃣ AddMaxRepsHistory
	err = repo.AddMaxRepsHistory(ctx, userID, 20)
	assert.NoError(t, err)

	// 1️⃣0️⃣ GetMaxRepsHistory
	history, err := repo.GetMaxRepsHistory(ctx, userID)
	assert.NoError(t, err)
	assert.True(t, len(history) > 0)
	assert.Equal(t, 20, history[0].MaxReps)

	// 1️⃣1️⃣ GetMaxRepsRecord
	record, err := repo.GetMaxRepsRecord(ctx, userID)
	assert.NoError(t, err)
	assert.Equal(t, 20, record.MaxReps)

	// 1️⃣2️⃣ GetFullStat
	fullStat, err := repo.GetFullStat(ctx, userID, today)
	assert.NoError(t, err)
	assert.Equal(t, 10, fullStat.TodayTotal)
	assert.NotEmpty(t, fullStat.Leaderboard)

	found := false
	for _, u := range fullStat.Leaderboard {
		if u.Username == username {
			found = true
			break
		}
	}

	assert.True(t, found)
}
