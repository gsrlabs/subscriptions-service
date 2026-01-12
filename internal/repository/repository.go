package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"subscription-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SubscriptionRepository defines the interface for managing subscription data in the storage.
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
	Update(ctx context.Context, sub *model.Subscription) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(
		ctx context.Context,
		userID *uuid.UUID,
		serviceName *string,
		limit, offset int,
	) ([]*model.Subscription, error)

	AggregateCost(
		ctx context.Context,
		userID *uuid.UUID,
		serviceName *string,
		from time.Time,
		to time.Time,
	) (int, error)
}

var (
	ErrNotFound = errors.New("subscription not found")
)

type subscriptionRepo struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepository creates a new instance of the subscription repository using a pgx connection pool.
func NewSubscriptionRepository(pool *pgxpool.Pool) SubscriptionRepository {
	return &subscriptionRepo{pool: pool}
}

// Create inserts a new subscription record into the database and populates the ID and timestamps.
func (r *subscriptionRepo) Create(ctx context.Context, sub *model.Subscription) error {
	log.Printf("INFO: creating subscription for user %s", sub.UserID)

	query := `
		INSERT INTO subscriptions (user_id, service_name, price, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(
		ctx,
		query,
		sub.UserID,
		sub.ServiceName,
		sub.Price,
		sub.StartDate,
		sub.EndDate,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		log.Printf("ERROR: failed to create subscription: %v", err)
		return err
	}

	log.Printf("INFO: subscription %s created", sub.ID)
	return nil
}

// GetByID retrieves a single subscription by its unique identifier. Returns ErrNotFound if no record exists.
func (r *subscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	log.Printf("INFO: getting subscription %s", id)

	query := `
		SELECT id, user_id, service_name, price, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	var sub model.Subscription
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.ServiceName,
		&sub.Price,
		&sub.StartDate,
		&sub.EndDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		log.Printf("WARN: subscription %s not found", id)
		return nil, ErrNotFound
	}

	if err != nil {
		log.Printf("ERROR: failed to get subscription %s: %v", id, err)
		return nil, err
	}

	return &sub, nil
}

// Update modifies an existing subscription record. Returns ErrNotFound if the subscription ID does not exist.
func (r *subscriptionRepo) Update(ctx context.Context, sub *model.Subscription) error {
	log.Printf("INFO: updating subscription %s", sub.ID)

	query := `
		UPDATE subscriptions
		SET service_name = $1,
			price = $2,
			start_date = $3,
			end_date = $4,
			updated_at = now()
		WHERE id = $5
	`

	cmd, err := r.pool.Exec(
		ctx,
		query,
		sub.ServiceName,
		sub.Price,
		sub.StartDate,
		sub.EndDate,
		sub.ID,
	)

	if err != nil {
		log.Printf("ERROR: failed to update subscription %s: %v", sub.ID, err)
		return err
	}

	if cmd.RowsAffected() == 0 {
		log.Printf("WARN: subscription %s not found for update", sub.ID)
		return ErrNotFound
	}

	return nil
}

// Delete removes a subscription record from the database by its ID. Returns ErrNotFound if no record was deleted.
func (r *subscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	log.Printf("INFO: deleting subscription %s", id)

	cmd, err := r.pool.Exec(
		ctx,
		`DELETE FROM subscriptions WHERE id = $1`,
		id,
	)

	if err != nil {
		log.Printf("ERROR: failed to delete subscription %s: %v", id, err)
		return err
	}

	if cmd.RowsAffected() == 0 {
		log.Printf("WARN: subscription %s not found for delete", id)
		return ErrNotFound
	}

	return nil
}

// List returns a slice of subscriptions based on optional filters (userID, serviceName) with pagination support.
func (r *subscriptionRepo) List(
	ctx context.Context,
	userID *uuid.UUID,
	serviceName *string,
	limit, offset int,
) ([]*model.Subscription, error) {

	log.Printf("INFO: listing subscriptions")

	query := `
		SELECT id, user_id, service_name, price, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND ($2::text IS NULL OR service_name = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.pool.Query(
		ctx,
		query,
		userID,
		serviceName,
		limit,
		offset,
	)
	if err != nil {
		log.Printf("ERROR: list subscriptions failed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var result []*model.Subscription

	for rows.Next() {
		var sub model.Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.ServiceName,
			&sub.Price,
			&sub.StartDate,
			&sub.EndDate,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &sub)
	}

	return result, nil
}

// AggregateCost calculates the total cost of active subscriptions for a given user and service within a specific time range.
func (r *subscriptionRepo) AggregateCost(
	ctx context.Context,
	userID *uuid.UUID,
	serviceName *string,
	from time.Time,
	to time.Time,
) (int, error) {

	log.Printf("INFO: aggregating subscriptions cost")

	query := `
		SELECT COALESCE(SUM(price), 0)
		FROM subscriptions
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND ($2::text IS NULL OR service_name = $2)
		  AND start_date <= $4
		  AND (end_date IS NULL OR end_date >= $3)
	`

	var total int
	err := r.pool.QueryRow(
		ctx,
		query,
		userID,
		serviceName,
		from,
		to,
	).Scan(&total)

	if err != nil {
		log.Printf("ERROR: aggregate cost failed: %v", err)
		return 0, err
	}

	log.Printf("INFO: aggregated cost = %d", total)
	return total, nil
}
