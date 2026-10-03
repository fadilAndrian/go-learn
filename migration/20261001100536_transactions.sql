-- +goose Up
CREATE TABLE transactions (
    id             BIGSERIAL       PRIMARY KEY,
    merchant_id    BIGINT          NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    reference_no   TEXT            NOT NULL UNIQUE,
    provider_ref   TEXT            UNIQUE,
    amount         NUMERIC(15, 2)  NOT NULL CHECK (amount > 0),
    status         TEXT            NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed')),
    payment_method TEXT,
    description    TEXT,
    additional_info JSONB,
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX transactions_merchant_id_idx ON transactions (merchant_id);

-- +goose Down
DROP TABLE transactions;
