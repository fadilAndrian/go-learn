-- +goose Up
CREATE TABLE api_logs (
    id             BIGSERIAL    PRIMARY KEY,
    transaction_id BIGINT       NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    -- type: inbound_create, outbound_create, inbound_check, outbound_check, callback
    type           TEXT         NOT NULL,
    method         TEXT         NOT NULL,
    url            TEXT         NOT NULL,
    request_body   JSONB,
    response_body  JSONB,
    status_code    INT,
    duration_ms    INT,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX api_logs_transaction_id_idx ON api_logs (transaction_id);

-- +goose Down
DROP TABLE api_logs;
