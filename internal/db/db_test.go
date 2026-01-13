package db

import (
	"context"
	"os"

	"subscription-service/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

// getTestConfig loads and returns configuration for testing.
func getTestConfig() *config.Config {
	if os.Getenv("DB_PASSWORD") == "" {
		os.Setenv("DB_PASSWORD", "password")
	}
	cfg, err := config.Load("../../config/config.yml")
	if err != nil {
		cfg, err = config.Load("config/config.yml")
		if err != nil {
			panic("failed to load config for tests: " + err.Error())
		}
	}

	// Apply settings for tests
	if cfg.Test.DBHost != "" {
		cfg.Database.Host = cfg.Test.DBHost
	} else {
		cfg.Database.Host = "localhost"
	}

	if cfg.Test.MigrationsPath != "" {
		cfg.Migrations.Path = cfg.Test.MigrationsPath
	} else {
		cfg.Migrations.Path = "../../migrations"
	}

	return cfg
}

// TestDatabaseConnectionAndMigrations verifies that the application can successfully
// connect to the database and that the migration tool (Goose) has initialized its version table.
func TestDatabaseConnectionAndMigrations(t *testing.T) {

	cfg := getTestConfig()

	ctx := context.Background()

	database, err := Connect(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	defer database.Pool.Close()

	assert.NoError(t, err)
	assert.NotNil(t, database)

	// Checking that the goose utility table exists
	var exists bool
	err = database.Pool.QueryRow(
		ctx,
		`SELECT EXISTS (
            SELECT 1 FROM information_schema.tables WHERE table_name = 'goose_db_version'
        )`,
	).Scan(&exists)

	assert.NoError(t, err)
	assert.True(t, exists, "goose_db_version table should exist")
}

// TestSubscriptionsTableExists confirms that the 'subscriptions' table was correctly
// created in the database schema after running migrations.
func TestSubscriptionsTableExists(t *testing.T) {

	cfg := getTestConfig()

	ctx := context.Background()
	database, err := Connect(ctx, cfg)
	assert.NoError(t, err)
	defer database.Pool.Close()

	var exists bool
	err = database.Pool.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_name = 'subscriptions'
		)
		`,
	).Scan(&exists)

	assert.NoError(t, err)
	assert.True(t, exists, "subscriptions table should exist")
}

// TestSubscriptionsIndexesExist ensures that all critical performance indexes
// defined in the migrations are present in the database.
func TestSubscriptionsIndexesExist(t *testing.T) {

	cfg := getTestConfig()

	ctx := context.Background()
	database, err := Connect(ctx, cfg)
	assert.NoError(t, err)
	defer database.Pool.Close()

	indexes := []string{
		"idx_subscriptions_user_id",
		"idx_subscriptions_service_name",
		"idx_subscriptions_dates",
		"idx_subscriptions_agg",
	}

	for _, idx := range indexes {
		var exists bool
		err := database.Pool.QueryRow(
			ctx,
			`
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE indexname = $1
			)
			`,
			idx,
		).Scan(&exists)

		assert.NoError(t, err)
		assert.True(t, exists, "index %s should exist", idx)
	}
}
