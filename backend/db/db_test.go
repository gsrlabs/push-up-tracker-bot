package db

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"trackerbot/config"

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

func TestConnect_Success(t *testing.T) {
	cfg := getTestConfig(t)
	ctx := context.Background()

	db, err := Connect(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)
	assert.NotNil(t, db.Pool)

	defer db.Pool.Close()

	// Проверяем ping
	err = db.Pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestConnect_NoAutoMigrations(t *testing.T) {
	cfg := getTestConfig(t)
	cfg.Migrations.Auto = false

	ctx := context.Background()

	db, err := Connect(ctx, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, db)

	defer db.Pool.Close()
}

func TestConnect_InvalidDSN(t *testing.T) {
	cfg := getTestConfig(t)
	cfg.Database.Host = "%%%invalid-host"

	ctx := context.Background()

	db, err := Connect(ctx, cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestRunMigrations_Success(t *testing.T) {
	cfg := getTestConfig(t)

	dsn := cfg.GetPGXConnConfig()

	err := runMigrations(dsn, "../migrations", true)
	assert.NoError(t, err)
}

func TestRunMigrations_InvalidPath(t *testing.T) {
	cfg := getTestConfig(t)

	dsn := cfg.GetPGXConnConfig()

	err := runMigrations(dsn, "invalid/path", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run migrations")
}

func TestMaskPassword_InvalidDSN(t *testing.T) {
	masked := maskPassword("%%%invalid%%%")
	assert.Equal(t, "***invalid dsn***", masked)
}

func TestGetPool(t *testing.T) {
	cfg := getTestConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Connect(ctx, cfg)
	assert.NoError(t, err)
	defer db.Pool.Close()

	pool := db.GetPool()
	assert.NotNil(t, pool)
}
