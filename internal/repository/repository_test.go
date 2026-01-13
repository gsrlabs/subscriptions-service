package repository_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"subscription-service/internal/config"
	"subscription-service/internal/db"
	"subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// setupTestDB initializes the test environment by loading configuration,
// establishing a database connection, and returning a cleanup function to truncate tables.
func setupTestDB(t *testing.T) (repository.SubscriptionRepository, func()) {

	cfg := getTestConfig()

	ctx := context.Background()
	database, err := db.Connect(ctx, cfg)
	require.NoError(t, err, "failed to connect to db")

	repo := repository.NewSubscriptionRepository(database.Pool)

	// Cleans up (called via defer in the test)
	cleanup := func() {
		_, err := database.Pool.Exec(ctx, "TRUNCATE subscriptions RESTART IDENTITY CASCADE")
		if err != nil {
			log.Printf("failed to truncate table: %v", err)
		}
		database.Pool.Close()
	}

	return repo, cleanup
}

// TestSubscriptionCRUD verifies the full lifecycle of a subscription (Create, Read, Update, Delete)
// using a real database connection.
func TestSubscriptionCRUD(t *testing.T) {

	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Data for the test
	userID := uuid.New()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	newSub := &model.Subscription{
		UserID:      userID,
		ServiceName: "Netflix",
		Price:       1000,
		StartDate:   startDate,
	}

	// 1. CREATE
	t.Run("Create", func(t *testing.T) {
		err := repo.Create(ctx, newSub)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, newSub.ID, "ID должен быть сгенерирован")
		assert.False(t, newSub.CreatedAt.IsZero())
	})

	// 2. GET
	t.Run("GetByID", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, newSub.ID)
		assert.NoError(t, err)
		assert.Equal(t, newSub.ID, fetched.ID)
		assert.Equal(t, "Netflix", fetched.ServiceName)
		assert.Equal(t, 1000, fetched.Price)
		// Check the date (UTC, without time, as the database has a date type)
		assert.Equal(t, startDate.Format("2006-01-02"), fetched.StartDate.Format("2006-01-02"))
	})

	// 3. UPDATE
	t.Run("Update", func(t *testing.T) {
		newSub.Price = 1200
		newSub.ServiceName = "Netflix Premium"

		err := repo.Update(ctx, newSub)
		assert.NoError(t, err)

		// Check through Get what has been updated
		fetched, err := repo.GetByID(ctx, newSub.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1200, fetched.Price)
		assert.Equal(t, "Netflix Premium", fetched.ServiceName)
	})

	// 4. DELETE
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, newSub.ID)
		assert.NoError(t, err)

		// Should get a NotFound error
		_, err = repo.GetByID(ctx, newSub.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

// TestListAndAggregation evaluates the repository's ability to filter records by various criteria
// and correctly sum subscription costs over specific time periods.
func TestListAndAggregation(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Creating test data
	subs := []*model.Subscription{
		// User 1
		{UserID: user1, ServiceName: "Yandex", Price: 300, StartDate: date(2025, 1, 1)},
		{UserID: user1, ServiceName: "Google", Price: 200, StartDate: date(2025, 2, 1)},
		// User 2
		{UserID: user2, ServiceName: "Yandex", Price: 300, StartDate: date(2025, 1, 1)},
	}

	for _, s := range subs {
		err := repo.Create(ctx, s)
		require.NoError(t, err)
	}

	t.Run("List Filter by UserID", func(t *testing.T) {
		list, err := repo.List(ctx, &user1, nil, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, list, 2, "У юзера 1 должно быть 2 подписки")
	})

	t.Run("List Filter by ServiceName", func(t *testing.T) {
		srvName := "Yandex"
		list, err := repo.List(ctx, nil, &srvName, 10, 0)
		assert.NoError(t, err)
		assert.Len(t, list, 2, "Всего 2 подписки на Яндекс")
	})

	t.Run("Aggregate Cost", func(t *testing.T) {
		// Calculate the amount for User1 for the period from January to March
		// Both subscriptions should be included (300 + 200 = 500)
		from := date(2025, 1, 1)
		to := date(2025, 3, 1)

		cost, err := repo.AggregateCost(ctx, &user1, nil, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 500, cost)
	})

	t.Run("Aggregate Cost Partial", func(t *testing.T) {
		// Calculate the amount for User1 only for January
		// Only the first subscription (300) should be included
		from := date(2025, 1, 1)
		to := date(2025, 1, 31)

		cost, err := repo.AggregateCost(ctx, &user1, nil, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 300, cost)
	})
}

// date is a test helper that returns a time.Time object for a given year, month, and day in UTC.
func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}
