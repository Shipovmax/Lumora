-- name: InsertPostIgnoreDuplicate :one
INSERT INTO posts (source_id, external_id, title, url, content, published_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (source_id, external_id) DO NOTHING
RETURNING *;
