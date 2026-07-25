-- +goose Up
CREATE TABLE user_context (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    content    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_context;
