-- +goose Up
CREATE TABLE events (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic         TEXT NOT NULL DEFAULT 'other',
    title         TEXT NOT NULL DEFAULT '',
    match_text    TEXT NOT NULL DEFAULT '',
    importance    INTEGER NOT NULL DEFAULT 0,
    source_count  INTEGER NOT NULL DEFAULT 0,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX events_last_seen_at_idx ON events (last_seen_at);

ALTER TABLE posts ADD COLUMN event_id UUID REFERENCES events (id) ON DELETE SET NULL;
CREATE INDEX posts_event_id_idx ON posts (event_id);

-- +goose Down
ALTER TABLE posts DROP COLUMN event_id;
DROP TABLE events;
