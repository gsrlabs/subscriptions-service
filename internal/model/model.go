package model

import (
    "fmt"
    "time"
    
    "github.com/google/uuid"
)

type Subscription struct {
    ID          uuid.UUID  `json:"id"`
    UserID      uuid.UUID  `json:"user_id"`
    ServiceName string     `json:"service_name"`
    Price       int        `json:"price"`
    StartDate   time.Time  `json:"start_date"`
    EndDate     *time.Time `json:"end_date,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateSubscriptionRequest - DTO для создания подписки
type CreateSubscriptionRequest struct {
    ServiceName string    `json:"service_name"`
    Price       int       `json:"price"`
    UserID      uuid.UUID `json:"user_id"`
    StartDate   string    `json:"start_date"` // Формат "MM-YYYY"
    EndDate     *string   `json:"end_date,omitempty"` // Формат "MM-YYYY"
}

// UpdateSubscriptionRequest - DTO для обновления подписки
type UpdateSubscriptionRequest struct {
    ServiceName *string    `json:"service_name,omitempty"`
    Price       *int       `json:"price,omitempty"`
    StartDate   *string    `json:"start_date,omitempty"` // Формат "MM-YYYY"
    EndDate     *string    `json:"end_date,omitempty"`   // Формат "MM-YYYY"
}

// ParseMonthYear парсит строку в формате "MM-YYYY" в time.Time
func ParseMonthYear(monthYear string) (time.Time, error) {
    parsedTime, err := time.Parse("01-2006", monthYear)
    if err != nil {
        return time.Time{}, fmt.Errorf("invalid date format, expected MM-YYYY: %w", err)
    }
    return parsedTime, nil
}

// ToTimePtr конвертирует строку в указатель на time.Time
func ToTimePtr(monthYear *string) (*time.Time, error) {
    if monthYear == nil {
        return nil, nil
    }
    
    parsedTime, err := ParseMonthYear(*monthYear)
    if err != nil {
        return nil, err
    }
    
    return &parsedTime, nil
}

// SubscriptionFilter - фильтр для поиска подписок
type SubscriptionFilter struct {
    UserID      *uuid.UUID
    ServiceName *string
    Limit       int
    Offset      int
}

// SummaryFilter - фильтр для агрегации
type SummaryFilter struct {
    UserID      *uuid.UUID
    ServiceName *string
    StartPeriod string // Формат "MM-YYYY"
    EndPeriod   string // Формат "MM-YYYY"
}
