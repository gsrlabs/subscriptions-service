package service_test

import (
	"context"

	"testing"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/service"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- 1. Создаем Mock Репозитория ---
// Это "фейковая" база данных. Она не ходит в Postgres,
// а просто возвращает то, что мы ей скажем в тесте.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, sub *model.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	args := m.Called(ctx, id)
	// Приводим первый аргумент к нужному типу, если он не nil
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Subscription), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, sub *model.Subscription) error {
	args := m.Called(ctx, sub)
	return args.Error(0)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, userID *uuid.UUID, serviceName *string, limit, offset int) ([]*model.Subscription, error) {
	args := m.Called(ctx, userID, serviceName, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Subscription), args.Error(1)
}

func (m *MockRepository) AggregateCost(ctx context.Context, userID *uuid.UUID, serviceName *string, from time.Time, to time.Time) (int, error) {
	args := m.Called(ctx, userID, serviceName, from, to)
	return args.Int(0), args.Error(1)
}

// --- 2. Пишем тесты ---

func TestCreateSubscription(t *testing.T) {
	// Подготовка данных
	mockRepo := new(MockRepository)
	svc := service.NewSubscriptionService(mockRepo)
	ctx := context.Background()
	uid := uuid.New()

	t.Run("Success", func(t *testing.T) {
		sub := &model.Subscription{
			UserID:      uid,
			ServiceName: "Netflix",
			Price:       100,
			StartDate:   time.Now(),
		}

		// Настраиваем Mock: "Когда вызовут Create с этим sub, верни nil (нет ошибки)"
		mockRepo.On("Create", ctx, sub).Return(nil)

		err := svc.Create(ctx, sub)

		assert.NoError(t, err)
		// Проверяем, что метод репозитория действительно был вызван
		mockRepo.AssertExpectations(t)
	})

	t.Run("Fail Validation Negative Price", func(t *testing.T) {
		sub := &model.Subscription{
			Price: -100, // Ошибка
		}

		// Mock настраивать не нужно, так как до репозитория дело не дойдет
		err := svc.Create(ctx, sub)

		assert.Error(t, err)
		assert.Equal(t, "price must be >= 0", err.Error())
		// Убеждаемся, что репозиторий НЕ вызывался
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("Fail Validation Dates", func(t *testing.T) {
		start := time.Now()
		end := start.Add(-24 * time.Hour) // Дата окончания раньше начала
		sub := &model.Subscription{
			Price:     100,
			StartDate: start,
			EndDate:   &end,
		}

		err := svc.Create(ctx, sub)

		assert.Error(t, err)
		assert.Equal(t, "end_date cannot be before start_date", err.Error())
		mockRepo.AssertNotCalled(t, "Create")
	})
}

func TestListSubscriptions(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewSubscriptionService(mockRepo)
	ctx := context.Background()

	t.Run("Default Limit/Offset Logic", func(t *testing.T) {
		// Мы передаем limit=0, offset=-1
		// Сервис должен превратить их в limit=20, offset=0 перед вызовом репозитория
		
		expectedList := []*model.Subscription{}
		
		// Ожидаем вызов с исправленными параметрами (20, 0)
		mockRepo.On("List", ctx, (*uuid.UUID)(nil), (*string)(nil), 20, 0).Return(expectedList, nil)

		res, err := svc.List(ctx, nil, nil, 0, -1)

		assert.NoError(t, err)
		assert.Equal(t, expectedList, res)
		mockRepo.AssertExpectations(t)
	})
}

func TestAggregate(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewSubscriptionService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		from := time.Now()
		to := from.Add(24 * time.Hour)
		
		// Мок должен вернуть 500 рублей
		mockRepo.On("AggregateCost", ctx, (*uuid.UUID)(nil), (*string)(nil), from, to).Return(500, nil)

		total, err := svc.Aggregate(ctx, nil, nil, from, to)
		
		assert.NoError(t, err)
		assert.Equal(t, 500, total)
	})

	t.Run("Invalid Period", func(t *testing.T) {
		from := time.Now()
		to := from.Add(-24 * time.Hour) // 'to' раньше 'from'

		total, err := svc.Aggregate(ctx, nil, nil, from, to)

		assert.ErrorIs(t, err, service.ErrInvalidPeriod)
		assert.Equal(t, 0, total)
		// Убеждаемся, что в базу запрос не пошел
		mockRepo.AssertNotCalled(t, "AggregateCost")
	})
}
