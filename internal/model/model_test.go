package model_test

import (
	"subscription-service/internal/model"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestValidateMonthYear checks the integrity of the custom validation logic.
// It ensures that:
//  1. Strings strictly follow the "MM-YYYY" format.
//  2. Out-of-range months (e.g., 13) are rejected.
//  3. Built-in validation tags like 'required' and 'min' (for ServiceName/Price)
//     work correctly in conjunction with custom tags.
func TestValidateMonthYear(t *testing.T) {
	tests := []struct {
		name    string
		request model.CreateSubscriptionRequest
		wantErr bool
	}{
		{
			name: "Success - Valid format",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Netflix",
				Price:       100,
				UserID:      uuid.New(),
				StartDate:   "01-2025",
			},
			wantErr: false,
		},
		{
			name: "Fail - Wrong format (YYYY-MM)",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Netflix",
				Price:       100,
				UserID:      uuid.New(),
				StartDate:   "2025-01",
			},
			wantErr: true,
		},
		{
			name: "Fail - Invalid month",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Netflix",
				Price:       100,
				UserID:      uuid.New(),
				StartDate:   "13-2025",
			},
			wantErr: true,
		},
		{
			name: "Success - EndDate is optional and valid",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Spotify",
				Price:       10,
				UserID:      uuid.New(),
				StartDate:   "05-2024",
				EndDate:     stringPtr("12-2025"),
			},
			wantErr: false,
		},
		{
			name: "Fail - ServiceName too short",
			request: model.CreateSubscriptionRequest{
				ServiceName: "A",
				Price:       100,
				UserID:      uuid.New(),
				StartDate:   "01-2025",
			},
			wantErr: true,
		},
		{
			name: "Success - January",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Test",
				Price:       1,
				UserID:      uuid.New(),
				StartDate:   "01-2025",
			},
			wantErr: false,
		},
		{
			name: "Fail - Invalid EndDate",
			request: model.CreateSubscriptionRequest{
				ServiceName: "Test",
				Price:       10,
				UserID:      uuid.New(),
				StartDate:   "01-2025",
				EndDate:     stringPtr("99-2025"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.Validate.Struct(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestToDomain verifies the transformation logic from a Request DTO to a Domain Model.
// It checks that date strings are correctly parsed into time.Time objects and
// that optional fields (like EndDate) are handled safely without nil pointer panics.
func TestToDomain(t *testing.T) {
	uid := uuid.New()

	t.Run("Success mapping", func(t *testing.T) {
		req := model.CreateSubscriptionRequest{
			ServiceName: "YouTube",
			Price:       200,
			UserID:      uid,
			StartDate:   "10-2024",
		}

		domain, err := model.ToDomain(req)

		assert.NoError(t, err)
		assert.Equal(t, "YouTube", domain.ServiceName)
		assert.Equal(t, 2024, domain.StartDate.Year())
		assert.Equal(t, time.October, domain.StartDate.Month())
		assert.Nil(t, domain.EndDate)
		assert.True(t, domain.StartDate.Day() == 1)

	})

	t.Run("Fail on invalid date parsing", func(t *testing.T) {
		req := model.CreateSubscriptionRequest{
			StartDate: "invalid",
		}
		_, err := model.ToDomain(req)
		assert.Error(t, err)
	})
}

// Helper for passing a string pointer
func stringPtr(s string) *string {
	return &s
}
