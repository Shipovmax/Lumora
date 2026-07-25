-- Кумулятивная схема БД (только CREATE, без goose Down) — источник типов для sqlc.
-- Реальные миграции выполняет goose из migrations/; этот файл нужно обновлять
-- в той же задаче, где добавляется новая goose-миграция, иначе sqlc generate
-- будет генерировать код по устаревшей схеме.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

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

CREATE TABLE user_context (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    content    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

CREATE TABLE posts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id    UUID NOT NULL REFERENCES sources (id) ON DELETE CASCADE,
    external_id  TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    url          TEXT NOT NULL DEFAULT '',
    content      TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (source_id, external_id)
);

CREATE INDEX posts_source_id_idx ON posts (source_id);

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
