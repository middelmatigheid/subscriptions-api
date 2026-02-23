-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_user_uuid ON subscriptions(user_uuid);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_subscriptions_service_name ON subscriptions(service_name);

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idx_subscriptions_user_uuid, idx_subscriptions_service_name;
-- +goose StatementEnd
