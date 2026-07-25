// Package repository — адаптер домена source к PostgreSQL поверх кода,
// сгенерированного sqlc (см. sqlc/gen). Реализует domain.Repository.
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shipovmax/Lumora/internal/source/domain"
	sqlcgen "github.com/Shipovmax/Lumora/internal/source/repository/sqlc/gen"
)

type Repository struct {
	q *sqlcgen.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlcgen.New(pool)}
}

var _ domain.Repository = (*Repository)(nil)

func (r *Repository) CreateSource(ctx context.Context, userID string, typ domain.Type, name, url string) (domain.Source, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return domain.Source{}, domain.ErrSourceNotFound
	}

	s, err := r.q.CreateSource(ctx, sqlcgen.CreateSourceParams{
		UserID: uid,
		Type:   string(typ),
		Name:   name,
		Url:    url,
	})
	if err != nil {
		return domain.Source{}, err
	}
	return toDomainSource(s), nil
}

func (r *Repository) ListSources(ctx context.Context, userID string) ([]domain.Source, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, domain.ErrSourceNotFound
	}

	rows, err := r.q.ListSources(ctx, uid)
	if err != nil {
		return nil, err
	}

	sources := make([]domain.Source, 0, len(rows))
	for _, row := range rows {
		sources = append(sources, toDomainSource(row))
	}
	return sources, nil
}

func (r *Repository) GetSource(ctx context.Context, userID, id string) (domain.Source, error) {
	uid, sid, err := parseUUIDPair(userID, id)
	if err != nil {
		return domain.Source{}, domain.ErrSourceNotFound
	}

	s, err := r.q.GetSource(ctx, sqlcgen.GetSourceParams{UserID: uid, ID: sid})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrSourceNotFound
		}
		return domain.Source{}, err
	}
	return toDomainSource(s), nil
}

func (r *Repository) GetSourceByID(ctx context.Context, id string) (domain.Source, error) {
	sid, err := parseUUID(id)
	if err != nil {
		return domain.Source{}, domain.ErrSourceNotFound
	}

	s, err := r.q.GetSourceByID(ctx, sid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrSourceNotFound
		}
		return domain.Source{}, err
	}
	return toDomainSource(s), nil
}

func (r *Repository) SetEnabled(ctx context.Context, userID, id string, enabled bool) (domain.Source, error) {
	uid, sid, err := parseUUIDPair(userID, id)
	if err != nil {
		return domain.Source{}, domain.ErrSourceNotFound
	}

	s, err := r.q.SetSourceEnabled(ctx, sqlcgen.SetSourceEnabledParams{UserID: uid, ID: sid, Enabled: enabled})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Source{}, domain.ErrSourceNotFound
		}
		return domain.Source{}, err
	}
	return toDomainSource(s), nil
}

func (r *Repository) DeleteSource(ctx context.Context, userID, id string) error {
	uid, sid, err := parseUUIDPair(userID, id)
	if err != nil {
		return domain.ErrSourceNotFound
	}

	rowsAffected, err := r.q.DeleteSource(ctx, sqlcgen.DeleteSourceParams{UserID: uid, ID: sid})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrSourceNotFound
	}
	return nil
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

func toDomainSource(s sqlcgen.Source) domain.Source {
	return domain.Source{
		ID:        uuidString(s.ID),
		UserID:    uuidString(s.UserID),
		Type:      domain.Type(s.Type),
		Name:      s.Name,
		URL:       s.Url,
		Enabled:   s.Enabled,
		CreatedAt: s.CreatedAt.Time,
		UpdatedAt: s.UpdatedAt.Time,
	}
}
