package model

import (
	"time"

	"github.com/google/uuid"
)

// Subscription — доменная модель подписки
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

// Data Transfer Object
// CreateSubscriptionRequest — входящий запрос
type CreateSubscriptionRequest struct {
	ServiceName string    `json:"service_name" validate:"required,min=2"`
	Price       int       `json:"price" validate:"required,min=0"`
	UserID      uuid.UUID `json:"user_id" validate:"required"`
	StartDate   string    `json:"start_date" validate:"required,mmYYYY"`
	EndDate     *string   `json:"end_date,omitempty" validate:"omitempty,mmYYYY"`
}

// SubscriptionResponse — ответ API
type SubscriptionResponse struct {
	ID          uuid.UUID `json:"id"`
	ServiceName string    `json:"service_name"`
	Price       int       `json:"price"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   string    `json:"start_date"`
	EndDate     *string   `json:"end_date,omitempty"`
}
