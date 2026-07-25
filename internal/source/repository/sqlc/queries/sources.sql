-- name: CreateSource :one
INSERT INTO sources (user_id, type, name, url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSources :many
SELECT * FROM sources
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetSource :one
SELECT * FROM sources
WHERE user_id = $1 AND id = $2;

-- name: SetSourceEnabled :one
UPDATE sources
SET enabled = $3, updated_at = now()
WHERE user_id = $1 AND id = $2
RETURNING *;

-- name: DeleteSource :execrows
DELETE FROM sources
WHERE user_id = $1 AND id = $2;
