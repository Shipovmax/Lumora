-- +goose Up
CREATE TABLE user_profiles (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    country    TEXT NOT NULL DEFAULT '',
    language   TEXT NOT NULL DEFAULT '',
    profession TEXT NOT NULL DEFAULT '',
    interests  TEXT[] NOT NULL DEFAULT '{}',
    topics     TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_profiles;
