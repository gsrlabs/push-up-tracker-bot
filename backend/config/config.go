// backend/config/config.go
package config

import (
	"fmt"
	"log"
	"os"

	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig       `mapstructure:"app"`
	Bot        BotConfig       `mapstructure:"bot"`
	Database   DatabaseConfig  `mapstructure:"database"`
	Migrations MigrationConfig `mapstructure:"migrations"`
	Test       TestConfig      `mapstructure:"test"`
}

type BotConfig struct {
	Token string `mapstructure:"token"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"` // будет из .env
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
}

type MigrationConfig struct {
	Path string `mapstructure:"path"`
	Auto bool   `mapstructure:"auto"`
}

type AppConfig struct {
	DebugMod bool   `mapstructure:"debug_mode"`
}

type TestConfig struct {
	DBHost         string `mapstructure:"db_host"`
	MigrationsPath string `mapstructure:"migrations_path"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.AutomaticEnv()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = v.BindEnv("bot.token", "TELEGRAM_BOT_TOKEN")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.name", "DB_NAME")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	// Проверка токена бота
	if c.Bot.Token == "" {
		// Пробуем получить из переменной окружения напрямую
		if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token == "" {
			return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
		}
	}

	// Проверка пароля БД
	if c.Database.Password == "" {
		if pwd := os.Getenv("DB_PASSWORD"); pwd == "" {
			return fmt.Errorf("DB_PASSWORD is required")
		}
	}

	// Проверка остальных параметров БД
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}

	// Проверка SSL режима
	validSSLModes := map[string]bool{
		"disable": true, "allow": true, "prefer": true,
		"require": true, "verify-ca": true, "verify-full": true,
	}
	if !validSSLModes[c.Database.SSLMode] {
		return fmt.Errorf("invalid sslmode: %s", c.Database.SSLMode)
	}

	// Проверка пула соединений
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("max_conns must be >= 1")
	}
	if c.Database.MinConns < 0 {
		return fmt.Errorf("min_conns cannot be negative")
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("min_conns cannot exceed max_conns")
	}

	return nil
}

// GetPGXConnConfig возвращает конфигурацию подключения для pgxpool
func (c *Config) GetPGXConnConfig() string {
	// Используем пароль из структуры или из env
	password := c.Database.Password
	if password == "" {
		password = os.Getenv("DB_PASSWORD")
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetBotToken возвращает токен бота (из .env)
func (c *Config) GetBotToken() string {
	if c.Bot.Token != "" {
		return c.Bot.Token
	}
	return os.Getenv("TELEGRAM_BOT_TOKEN")
}

// GetConfig загружает конфигурацию, перебирая возможные пути
func GetConfig() (*Config, error) {

	// Возможные пути для поиска
	configPaths := []string{
		"/config/config.yml",         // Docker
		"../../../config/config.yml", // Глубоко в структуре
		"../../config/config.yml",    // На уровень глубже
		"../config/config.yml",       // В родительской директории
		"./config/config.yml",        // В текущей директории
		"config/config.yml",          // Относительный путь
		"./config.yaml",              // YAML в текущей
		"config.yaml",                // YAML относительно
	}

	var lastErr error

	// Перебираем все пути
	for _, p := range configPaths {
		cfg, err := Load(p)
		if err == nil {
			log.Printf("INFO: loaded config from %s", p)
			return cfg, nil
		}
		lastErr = err
	}

	// Если ничего не нашли, возвращаем ошибку
	return nil, lastErr
}
