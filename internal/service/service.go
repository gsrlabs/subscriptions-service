package service

import (
	"context"
	"errors"
	"log"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/repository"

	"github.com/google/uuid"
)

// SubscriptionService defines the business logic operations for managing subscriptions.
type SubscriptionService interface {
	Create(ctx context.Context, sub *model.Subscription) error
	Get(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
	Update(ctx context.Context, sub *model.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(
		ctx context.Context,
		userID *uuid.UUID,
		serviceName *string,
		limit, offset int,
	) ([]*model.Subscription, error)

	Aggregate(
		ctx context.Context,
		userID *uuid.UUID,
		serviceName *string,
		from time.Time,
		to time.Time,
	) (int, error)
}

var (
	ErrInvalidPeriod = errors.New("invalid aggregation period")
)

type subscriptionService struct {
	repo repository.SubscriptionRepository
}

// NewSubscriptionService creates a new instance of the subscription service with the given repository.
func NewSubscriptionService(repo repository.SubscriptionRepository) SubscriptionService {
	return &subscriptionService{repo: repo}
}

// Create validates and saves a new subscription.
// It returns an error if the price is negative or if the end date is before the start date.
func (s *subscriptionService) Create(ctx context.Context, sub *model.Subscription) error {
	log.Printf("INFO: service create subscription for user %s", sub.UserID)

	if sub.Price < 0 {
		log.Printf("ERROR: negative price")
		return errors.New("price must be >= 0")
	}

	if sub.EndDate != nil && sub.EndDate.Before(sub.StartDate) {
		log.Printf("ERROR: end_date before start_date")
		return errors.New("end_date cannot be before start_date")
	}

	err := s.repo.Create(ctx, sub)
	if err != nil {
		log.Printf("ERROR: repository create failed: %v", err)
		return err
	}

	log.Printf("INFO: subscription created: %s", sub.ID)
	return nil
}

// Get retrieves a subscription by its ID from the repository.
func (s *subscriptionService) Get(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	log.Printf("INFO: service get subscription %s", id)

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("ERROR: get subscription failed: %v", err)
		return nil, err
	}

	return sub, nil
}

// Update validates and updates an existing subscription.
// It enforces the same validation rules as the Create method (price and dates).
func (s *subscriptionService) Update(ctx context.Context, sub *model.Subscription) error {
	log.Printf("INFO: service update subscription %s", sub.ID)

	if sub.Price < 0 {
		return errors.New("price must be >= 0")
	}

	if sub.EndDate != nil && sub.EndDate.Before(sub.StartDate) {
		return errors.New("end_date cannot be before start_date")
	}

	err := s.repo.Update(ctx, sub)
	if err != nil {
		log.Printf("ERROR: repository update failed: %v", err)
		return err
	}

	log.Printf("INFO: subscription updated %s", sub.ID)
	return nil
}

// Delete removes a subscription record via the repository.
func (s *subscriptionService) Delete(ctx context.Context, id uuid.UUID) error {
	log.Printf("INFO: service delete subscription %s", id)

	err := s.repo.Delete(ctx, id)
	if err != nil {
		log.Printf("ERROR: delete failed: %v", err)
		return err
	}

	log.Printf("INFO: subscription deleted %s", id)
	return nil
}

// List fetches a collection of subscriptions with default values for pagination (limit: 20, offset: 0)
// if they are not provided or invalid.
func (s *subscriptionService) List(
	ctx context.Context,
	userID *uuid.UUID,
	serviceName *string,
	limit, offset int,
) ([]*model.Subscription, error) {

	log.Printf("INFO: service list subscriptions")

	if limit <= 0 {
		limit = 20
	}

	if offset < 0 {
		offset = 0
	}

	return s.repo.List(ctx, userID, serviceName, limit, offset)
}

// Aggregate calculates the total cost of subscriptions for a specific period.
// It returns ErrInvalidPeriod if the start time (from) is after the end time (to).
func (s *subscriptionService) Aggregate(
	ctx context.Context,
	userID *uuid.UUID,
	serviceName *string,
	from time.Time,
	to time.Time,
) (int, error) {

	log.Printf("INFO: service aggregate subscriptions")

	if from.After(to) {
		log.Printf("ERROR: invalid aggregation period")
		return 0, ErrInvalidPeriod
	}

	total, err := s.repo.AggregateCost(ctx, userID, serviceName, from, to)
	if err != nil {
		log.Printf("ERROR: aggregation failed: %v", err)
		return 0, err
	}

	log.Printf("INFO: aggregation result = %d", total)
	return total, nil
}
