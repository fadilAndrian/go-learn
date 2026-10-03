-- +goose Up
CREATE TABLE merchants (
    id          BIGSERIAL       PRIMARY KEY,
    name        TEXT            NOT NULL,
    phone       TEXT,
    address     TEXT,
    description TEXT,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- +goose Down
DROP TABLE merchants;
