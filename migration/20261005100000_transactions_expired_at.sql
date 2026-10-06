-- +goose Up
ALTER TABLE transactions ADD COLUMN expired_at TIMESTAMPTZ;
CREATE INDEX transactions_pending_idx ON transactions (created_at) WHERE status = 'pending';

-- +goose Down
DROP INDEX transactions_pending_idx;
ALTER TABLE transactions DROP COLUMN expired_at;
