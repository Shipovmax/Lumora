package domain

import "context"

// Repository — порт хранения сгенерированных объяснений. Реализуется в
// internal/ai/repository поверх Postgres/sqlc.
type Repository interface {
	// SaveExplanation сохраняет объяснение (upsert по паре event_id+user_id —
	// повторная генерация для той же пары заменяет предыдущий результат).
	SaveExplanation(ctx context.Context, exp Explanation) (Explanation, error)
	GetExplanation(ctx context.Context, eventID, userID string) (Explanation, error)
}
