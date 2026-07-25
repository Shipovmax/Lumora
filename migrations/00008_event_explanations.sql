-- +goose Up
CREATE TABLE event_explanations (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id                UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    user_id                 UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    what_happened           TEXT NOT NULL DEFAULT '',
    why_it_happened         TEXT NOT NULL DEFAULT '',
    what_changed            TEXT NOT NULL DEFAULT '',
    what_it_means_for_user  TEXT NOT NULL DEFAULT '',
    model                   TEXT NOT NULL DEFAULT '',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id)
);

CREATE INDEX event_explanations_user_id_idx ON event_explanations (user_id);

-- +goose Down
DROP TABLE event_explanations;
