package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents the core domain model for a user's service subscription.
type Subscription struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ServiceName string
	Price       int
	StartDate   time.Time
	EndDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateSubscriptionRequest defines the schema for incoming subscription creation or update data.
// It includes validation tags for business rules like minimum price and date formats.
type CreateSubscriptionRequest struct {
	ServiceName string    `json:"service_name" validate:"required,min=2" extensions:"x-order=1"`
	Price       int       `json:"price" validate:"required,min=0" extensions:"x-order=2"`
	UserID      uuid.UUID `json:"user_id" validate:"required" extensions:"x-order=3"`
	StartDate   string    `json:"start_date" validate:"required,mmYYYY" extensions:"x-order=4"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,mmYYYY" extensions:"x-order=5"`
}

// SubscriptionResponse represents the data structure returned to API clients.
// It uses strings for dates to ensure consistent formatting across different platforms.
type SubscriptionResponse struct {
	ID          uuid.UUID `json:"id" extensions:"x-order=1"`
	ServiceName string    `json:"service_name" extensions:"x-order=2"`
	Price       int       `json:"price" extensions:"x-order=3"`
	UserID      uuid.UUID `json:"user_id" extensions:"x-order=4"`
	StartDate   string    `json:"start_date" extensions:"x-order=5"`
	EndDate     *string   `json:"end_date,omitempty" extensions:"x-order=6"`
}
