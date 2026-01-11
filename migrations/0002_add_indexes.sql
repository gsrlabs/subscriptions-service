-- +goose Up
CREATE INDEX idx_subscriptions_user_id
    ON subscriptions(user_id);

CREATE INDEX idx_subscriptions_service_name
    ON subscriptions(service_name);

CREATE INDEX idx_subscriptions_dates
    ON subscriptions(start_date, end_date);

CREATE INDEX idx_subscriptions_agg
    ON subscriptions(user_id, service_name, start_date, end_date);

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_user_id;
DROP INDEX IF EXISTS idx_subscriptions_service_name;
DROP INDEX IF EXISTS idx_subscriptions_dates;
DROP INDEX IF EXISTS idx_subscriptions_agg;
