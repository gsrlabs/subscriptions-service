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

func NewSubscriptionService(repo repository.SubscriptionRepository) SubscriptionService {
	return &subscriptionService{repo: repo}
}

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

func (s *subscriptionService) Get(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	log.Printf("INFO: service get subscription %s", id)

	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Printf("ERROR: get subscription failed: %v", err)
		return nil, err
	}

	return sub, nil
}

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


