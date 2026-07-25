-- name: GetOrCreateContext :one
INSERT INTO user_context (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: UpsertContext :one
INSERT INTO user_context (user_id, content)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET
    content = EXCLUDED.content,
    updated_at = now()
RETURNING *;
