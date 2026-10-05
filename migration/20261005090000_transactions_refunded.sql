-- +goose Up
ALTER TABLE transactions DROP CONSTRAINT transactions_status_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_status_check
    CHECK (status IN ('pending', 'paid', 'failed', 'refunded'));

-- +goose Down
UPDATE transactions SET status = 'paid' WHERE status = 'refunded';
ALTER TABLE transactions DROP CONSTRAINT transactions_status_check;
ALTER TABLE transactions ADD CONSTRAINT transactions_status_check
    CHECK (status IN ('pending', 'paid', 'failed'));
