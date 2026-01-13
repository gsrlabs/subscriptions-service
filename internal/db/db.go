package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"subscription-service/internal/config"
	"time"

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

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	pgcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pgx config: %w", err)
	}

	pgcfg.MaxConns = cfg.Database.MaxConns
	pgcfg.MinConns = cfg.Database.MinConns
	pgcfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, pgcfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	// Checking the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Printf("INFO: connected to database")

	if err := runMigrations(dsn, cfg.Migrations.Path); err != nil {
		return nil, err
	}

	return &Database{Pool: pool}, nil
}

// runMigrations applies database schema changes using the goose provider
// from the specified migrations directory.
func runMigrations(dsn string, migrationsPath string) error {

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Printf("INFO: database migrations applied")
	return nil
}
