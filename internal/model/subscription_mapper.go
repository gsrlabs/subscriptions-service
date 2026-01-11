package model

import "time"

// ToDomain преобразует CreateSubscriptionRequest в Subscription
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
