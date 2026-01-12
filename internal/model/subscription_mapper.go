package model

import "time"

// ToDomain transforms a CreateSubscriptionRequest into a Subscription domain model.
// It parses date strings from the "MM-YYYY" format into time.Time objects.
func ToDomain(req CreateSubscriptionRequest) (*Subscription, error) {
	startDate, err := time.Parse("01-2006", req.StartDate)
	if err != nil {
		return nil, err
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("01-2006", *req.EndDate)
		if err != nil {
			return nil, err
		}
		endDate = &parsed
	}

	return &Subscription{
		UserID:      req.UserID,
		ServiceName: req.ServiceName,
		Price:       req.Price,
		StartDate:   startDate,
		EndDate:     endDate,
	}, nil
}

// ToResponse converts a Subscription domain model into a SubscriptionResponse DTO.
// It formats time.Time objects back into "MM-YYYY" strings for API consumers.
func ToResponse(sub *Subscription) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:          sub.ID,
		ServiceName: sub.ServiceName,
		Price:       sub.Price,
		UserID:      sub.UserID,
		StartDate:   sub.StartDate.Format("01-2006"),
	}

	if sub.EndDate != nil {
		end := sub.EndDate.Format("01-2006")
		resp.EndDate = &end
	}

	return resp
}
