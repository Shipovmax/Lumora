-- +goose Up
CREATE TABLE briefings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type         TEXT NOT NULL CHECK (type IN ('morning', 'evening')),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX briefings_user_id_idx ON briefings (user_id);

CREATE TABLE briefing_events (
    briefing_id UUID NOT NULL REFERENCES briefings (id) ON DELETE CASCADE,
    event_id    UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    rank        INTEGER NOT NULL,
    PRIMARY KEY (briefing_id, event_id)
);

CREATE INDEX briefing_events_event_id_idx ON briefing_events (event_id);

-- +goose Down
DROP TABLE briefing_events;
DROP TABLE briefings;
