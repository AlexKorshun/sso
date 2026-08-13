-- +goose Up
CREATE TABLE users
(
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    pass_hash BYTEA NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
