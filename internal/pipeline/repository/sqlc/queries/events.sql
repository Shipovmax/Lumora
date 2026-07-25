-- name: GetPostsByID :many
SELECT * FROM posts
WHERE id = ANY($1::uuid[]);

-- name: ListRecentEvents :many
SELECT * FROM events
WHERE last_seen_at >= $1
ORDER BY last_seen_at DESC;

-- name: GetEventByID :one
SELECT * FROM events
WHERE id = $1;

-- name: CreateEvent :one
INSERT INTO events (topic, title, match_text, importance, source_count, first_seen_at, last_seen_at)
VALUES ($1, $2, $3, 20, 1, $4, $4)
RETURNING *;

-- name: AssignPostToEvent :exec
UPDATE posts SET event_id = $1
WHERE id = $2;

-- name: RecomputeEventStats :one
UPDATE events e
SET source_count = sub.source_count,
    importance = LEAST(100, sub.source_count * 20),
    last_seen_at = GREATEST(e.last_seen_at, $2),
    match_text = $3,
    updated_at = now()
FROM (
    SELECT COUNT(DISTINCT p.source_id) AS source_count
    FROM posts p
    WHERE p.event_id = $1
) sub
WHERE e.id = $1
RETURNING e.*;
