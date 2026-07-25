-- name: UpsertExplanation :one
INSERT INTO event_explanations (event_id, user_id, what_happened, why_it_happened, what_changed, what_it_means_for_user, model)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (event_id, user_id) DO UPDATE SET
    what_happened = EXCLUDED.what_happened,
    why_it_happened = EXCLUDED.why_it_happened,
    what_changed = EXCLUDED.what_changed,
    what_it_means_for_user = EXCLUDED.what_it_means_for_user,
    model = EXCLUDED.model,
    updated_at = now()
RETURNING *;

-- name: GetExplanation :one
SELECT * FROM event_explanations
WHERE event_id = $1 AND user_id = $2;
