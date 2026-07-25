// Package repository — адаптер домена ai к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/ai/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/ai/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) SaveExplanation(ctx context.Context, exp domain.Explanation) (domain.Explanation, error) {
	eventID, userID, err := parseUUIDPair(exp.EventID, exp.UserID)
	if err != nil {
		return domain.Explanation{}, domain.ErrExplanationNotFound
	}

	e, err := r.q.UpsertExplanation(ctx, sqlcgen.UpsertExplanationParams{
		EventID:            eventID,
		UserID:             userID,
		WhatHappened:       exp.WhatHappened,
		WhyItHappened:      exp.WhyItHappened,
		WhatChanged:        exp.WhatChanged,
		WhatItMeansForUser: exp.WhatItMeansForUser,
		Model:              exp.Model,
	})
	if err != nil {
		return domain.Explanation{}, err
	}
	return toDomainExplanation(e), nil
}

func (r *Repository) GetExplanation(ctx context.Context, eventID, userID string) (domain.Explanation, error) {
	eid, uid, err := parseUUIDPair(eventID, userID)
	if err != nil {
		return domain.Explanation{}, domain.ErrExplanationNotFound
	}

	e, err := r.q.GetExplanation(ctx, sqlcgen.GetExplanationParams{EventID: eid, UserID: uid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Explanation{}, domain.ErrExplanationNotFound
		}
		return domain.Explanation{}, err
	}
	return toDomainExplanation(e), nil
}

func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func parseUUIDPair(a, b string) (pgtype.UUID, pgtype.UUID, error) {
	ua, err := parseUUID(a)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	ub, err := parseUUID(b)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return ua, ub, nil
}

func uuidString(id pgtype.UUID) string {
	v, err := id.Value()
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func toDomainExplanation(e sqlcgen.EventExplanation) domain.Explanation {
	return domain.Explanation{
		ID:                 uuidString(e.ID),
		EventID:            uuidString(e.EventID),
		UserID:             uuidString(e.UserID),
		WhatHappened:       e.WhatHappened,
		WhyItHappened:      e.WhyItHappened,
		WhatChanged:        e.WhatChanged,
		WhatItMeansForUser: e.WhatItMeansForUser,
		Model:              e.Model,
		CreatedAt:          e.CreatedAt.Time,
		UpdatedAt:          e.UpdatedAt.Time,
	}
}
