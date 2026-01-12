package db

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

// TestMain sets up the test environment by loading environment variables and
// overriding database configuration specifically for integration testing.
func TestMain(m *testing.M) {

	envFile := "../../.env"
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("INFO: %s not found, using environment variables", envFile)
	}

	// REDEFINE variables for tests.
	// If there are test settings in .env, replace the main ones with them.

	// Replace the host (postgres -> localhost)
	if testHost := os.Getenv("DB_HOST_TEST"); testHost != "" {
		os.Setenv("DB_HOST", testHost)
	}

	// Replace the path to migrations (./migrations -> ../../migrations)
	if testPath := os.Getenv("MIGRATION_PATH_TEST"); testPath != "" {
		os.Setenv("MIGRATION_PATH", testPath)
	}

	// Running tests
	code := m.Run()

	// (Optional) You can clear the variables, but the process still ends
	os.Exit(code)
}

// TestDatabaseConnectionAndMigrations verifies that the application can successfully
// connect to the database and that the migration tool (Goose) has initialized its version table.
func TestDatabaseConnectionAndMigrations(t *testing.T) {
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB env variables are not set")
	}

	ctx := context.Background()

	database, err := Connect(ctx)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	defer database.Pool.Close()

	assert.NoError(t, err)
	assert.NotNil(t, database)

	// Проверяем, что служебная таблица goose существует
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
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB env variables are not set")
	}

	ctx := context.Background()
	database, err := Connect(ctx)
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
	if os.Getenv("DB_HOST") == "" {
		t.Skip("DB env variables are not set")
	}

	ctx := context.Background()
	database, err := Connect(ctx)
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
