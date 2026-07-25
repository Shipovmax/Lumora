-- +goose Up
CREATE TABLE sources (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('rss', 'youtube', 'telegram')),
    name       TEXT NOT NULL,
    url        TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sources_user_id_idx ON sources (user_id);

-- +goose Down
DROP TABLE sources;
