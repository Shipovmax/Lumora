-- name: ListCandidateEvents :many
SELECT DISTINCT e.id, e.topic, e.title, e.importance, e.last_seen_at
FROM events e
JOIN posts p ON p.event_id = e.id
JOIN sources s ON s.id = p.source_id
WHERE s.user_id = $1
  AND e.last_seen_at >= $2
  AND NOT EXISTS (
      SELECT 1 FROM briefing_events be
      JOIN briefings b ON b.id = be.briefing_id
      WHERE be.event_id = e.id AND b.user_id = $1
  )
ORDER BY e.importance DESC, e.last_seen_at DESC
LIMIT $3;

-- name: CreateBriefing :one
INSERT INTO briefings (user_id, type)
VALUES ($1, $2)
RETURNING *;

-- name: AddBriefingEvent :exec
INSERT INTO briefing_events (briefing_id, event_id, rank)
VALUES ($1, $2, $3);

-- name: ListActiveUserIDs :many
SELECT DISTINCT user_id FROM sources;
