-- name: GetOrCreateProfile :one
INSERT INTO user_profiles (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: UpsertProfile :one
INSERT INTO user_profiles (user_id, name, country, language, profession, interests, topics)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    name = EXCLUDED.name,
    country = EXCLUDED.country,
    language = EXCLUDED.language,
    profession = EXCLUDED.profession,
    interests = EXCLUDED.interests,
    topics = EXCLUDED.topics,
    updated_at = now()
RETURNING *;
