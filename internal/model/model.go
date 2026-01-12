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
	ServiceName string    `json:"service_name" validate:"required,min=2"`
	Price       int       `json:"price" validate:"required,min=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required,mmYYYY"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,mmYYYY"`
}

// SubscriptionResponse represents the data structure returned to API clients.
// It uses strings for dates to ensure consistent formatting across different platforms.
type SubscriptionResponse struct {
	ID          uuid.UUID `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
}
