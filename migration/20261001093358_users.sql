-- +goose Up
CREATE TABLE users (
	id         	BIGSERIAL 	PRIMARY KEY,
	merchant_id BIGINT      REFERENCES merchants(id) ON DELETE CASCADE,
	name       	TEXT        NOT NULL,
	email      	TEXT        NOT NULL,
	password   	TEXT        NOT NULL,
	created_at 	TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at 	TIMESTAMPTZ NOT NULL DEFAULT now(),
	deleted_at 	TIMESTAMPTZ
);

-- email harus unik hanya untuk user yang belum dihapus
CREATE UNIQUE INDEX users_email_active_idx ON users (email) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE users;
