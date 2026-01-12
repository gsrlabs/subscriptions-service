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

// MockRepository is a mock implementation of the SubscriptionRepository interface.
// It is used to simulate database behavior and verify calls from the service layer.
type MockRepository struct {
	mock.Mock
}

// Mocks

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

// TestCreateSubscription verifies the service-level validation for new subscriptions,
// ensuring that records are only saved if price and dates are valid.
func TestCreateSubscription(t *testing.T) {
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

		// Setting up Mock: "When Create is called with this sub, return nil (no error)"
		mockRepo.On("Create", ctx, sub).Return(nil)

		err := svc.Create(ctx, sub)

		assert.NoError(t, err)
		// Check that the repository method was actually called
		mockRepo.AssertExpectations(t)
	})

	t.Run("Fail Validation Negative Price", func(t *testing.T) {
		sub := &model.Subscription{
			Price: -100, // error
		}

		// No need to configure the mock, as it won't reach the repository:)
		err := svc.Create(ctx, sub)

		assert.Error(t, err)
		assert.Equal(t, "price must be >= 0", err.Error())
		// Make sure that the repository was NOT called
		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("Fail Validation Dates", func(t *testing.T) {
		start := time.Now()
		end := start.Add(-24 * time.Hour) // End date before start
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

// TestListSubscriptions checks the service logic for handling pagination parameters,
// specifically the assignment of default values for invalid limit and offset inputs.
func TestListSubscriptions(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewSubscriptionService(mockRepo)
	ctx := context.Background()

	t.Run("Default Limit/Offset Logic", func(t *testing.T) {
		// We pass limit=0, offset=-1
		// The service should turn them into limit=20, offset=0 before calling the repository

		expectedList := []*model.Subscription{}

		// Expecting a call with corrected parameters (20, 0)
		mockRepo.On("List", ctx, (*uuid.UUID)(nil), (*string)(nil), 20, 0).Return(expectedList, nil)

		res, err := svc.List(ctx, nil, nil, 0, -1)

		assert.NoError(t, err)
		assert.Equal(t, expectedList, res)
		mockRepo.AssertExpectations(t)
	})
}

// TestAggregate ensures the cost calculation logic correctly handles date ranges
// and prevents repository calls when the aggregation period is invalid.
func TestAggregate(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := service.NewSubscriptionService(mockRepo)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		from := time.Now()
		to := from.Add(24 * time.Hour)

		// Mok must return 500 rubles
		mockRepo.On("AggregateCost", ctx, (*uuid.UUID)(nil), (*string)(nil), from, to).Return(500, nil)

		total, err := svc.Aggregate(ctx, nil, nil, from, to)

		assert.NoError(t, err)
		assert.Equal(t, 500, total)
	})

	t.Run("Invalid Period", func(t *testing.T) {
		from := time.Now()
		to := from.Add(-24 * time.Hour) // 'to' before 'from'

		total, err := svc.Aggregate(ctx, nil, nil, from, to)

		assert.ErrorIs(t, err, service.ErrInvalidPeriod)
		assert.Equal(t, 0, total)
		// Make sure that the request is not sent to the database
		mockRepo.AssertNotCalled(t, "AggregateCost")
	})
}
