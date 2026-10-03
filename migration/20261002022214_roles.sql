-- +goose Up
CREATE TABLE roles (
    id          BIGSERIAL   PRIMARY KEY,
    name        TEXT        NOT NULL,
    guard       TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- +goose Down
DROP TABLE roles;
