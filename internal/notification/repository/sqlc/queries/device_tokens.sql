-- name: RegisterDevice :one
INSERT INTO device_tokens (user_id, platform, token)
VALUES ($1, $2, $3)
ON CONFLICT (token) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    updated_at = now()
RETURNING *;

-- name: ListDeviceTokens :many
SELECT * FROM device_tokens
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: RemoveDeviceToken :exec
DELETE FROM device_tokens
WHERE token = $1;
