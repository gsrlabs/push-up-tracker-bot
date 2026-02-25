package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"
	"trackerbot/config"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Database wraps the pgxpool.Pool to provide a unified database access point.
type Database struct {
	Pool *pgxpool.Pool
}

// Connect establishes a connection pool to PostgreSQL using environment variables
// and automatically executes pending migrations.
func Connect(ctx context.Context, cfg *config.Config) (*Database, error) {

	dsn := cfg.GetPGXConnConfig()
	log.Printf("Connecting to database with DSN: %s", maskPassword(dsn))

	pgcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	pgcfg.MaxConns = cfg.Database.MaxConns
	pgcfg.MinConns = cfg.Database.MinConns
	pgcfg.MaxConnLifetime = 30 * time.Minute

	if cfg.Migrations.Auto {
		if err := runMigrations(dsn, cfg.Migrations.Path, cfg.Migrations.Auto); err != nil {
			return nil, err
		}
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgcfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Println("connected to database")
	return &Database{Pool: pool}, nil
}

// runMigrations applies database schema changes using the goose provider
// from the specified migrations directory.
func runMigrations(dsn, migrationsPath string, mode bool) error {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			log.Fatalf("error close database %v", err)

		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	log.Println("running migrations")

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Println("migrations finished successfully")
	return nil
}

func maskPassword(dsn string) string {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "***invalid dsn***"
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		parsed.User = url.UserPassword(username, "***")
	}

	return parsed.String()
}

// GetPool returns the underlying connection pool
func (d *Database) GetPool() *pgxpool.Pool {
	return d.Pool
}
