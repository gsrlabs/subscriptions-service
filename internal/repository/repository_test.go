package repository_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"subscription-service/internal/db"
	"subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB инициализирует соединение с БД и возвращает репозиторий + функцию очистки
func setupTestDB(t *testing.T) (repository.SubscriptionRepository, func()) {
	// 1. Загружаем env (для локального запуска)
	_ = godotenv.Load("../../.env")

	// Настройка путей для тестов, если запускаем из этой папки
	if os.Getenv("MIGRATION_PATH_TEST") != "" {
		os.Setenv("MIGRATION_PATH", os.Getenv("MIGRATION_PATH_TEST"))
	} else {
		// Fallback если переменная не задана (локальный запуск go test ./...)
		os.Setenv("MIGRATION_PATH", "../../migrations")
	}
    
    // Подменяем хост если есть тестовая переменная
    if host := os.Getenv("DB_HOST_TEST"); host != "" {
        os.Setenv("DB_HOST", host)
    }

	ctx := context.Background()
	database, err := db.Connect(ctx)
	require.NoError(t, err, "failed to connect to db")

	repo := repository.NewSubscriptionRepository(database.Pool)

	// Функция очистки (вызывается через defer в тесте)
	cleanup := func() {
		_, err := database.Pool.Exec(ctx, "TRUNCATE subscriptions RESTART IDENTITY CASCADE")
		if err != nil {
			log.Printf("failed to truncate table: %v", err)
		}
		database.Pool.Close()
	}

	return repo, cleanup
}

func TestSubscriptionCRUD(t *testing.T) {
	// Подготовка
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Данные для теста
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
		// Проверяем дату (UTC, без времени, так как в БД тип date)
		assert.Equal(t, startDate.Format("2006-01-02"), fetched.StartDate.Format("2006-01-02"))
	})

	// 3. UPDATE
	t.Run("Update", func(t *testing.T) {
		newSub.Price = 1200
		newSub.ServiceName = "Netflix Premium"
		
		err := repo.Update(ctx, newSub)
		assert.NoError(t, err)

		// Проверяем через Get, что обновилось
		fetched, err := repo.GetByID(ctx, newSub.ID)
		assert.NoError(t, err)
		assert.Equal(t, 1200, fetched.Price)
		assert.Equal(t, "Netflix Premium", fetched.ServiceName)
	})

	// 4. DELETE
	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, newSub.ID)
		assert.NoError(t, err)

		// Должны получить ошибку NotFound
		_, err = repo.GetByID(ctx, newSub.ID)
		assert.ErrorIs(t, err, repository.ErrNotFound)
	})
}

func TestListAndAggregation(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	user1 := uuid.New()
	user2 := uuid.New()

	// Создаем тестовые данные
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
		// Считаем сумму для User1 за период с Января по Март
		// Должны попасть обе подписки (300 + 200 = 500)
		from := date(2025, 1, 1)
		to := date(2025, 3, 1)

		cost, err := repo.AggregateCost(ctx, &user1, nil, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 500, cost)
	})

	t.Run("Aggregate Cost Partial", func(t *testing.T) {
		// Считаем сумму для User1 только за Январь
		// Должна попасть только первая подписка (300)
		from := date(2025, 1, 1)
		to := date(2025, 1, 31)

		cost, err := repo.AggregateCost(ctx, &user1, nil, from, to)
		assert.NoError(t, err)
		assert.Equal(t, 300, cost)
	})
}

// Хелпер для быстрого создания даты
func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}
