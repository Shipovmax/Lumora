// Package service реализует бизнес-логику домена source: добавление,
// просмотр, включение/выключение и удаление источников пользователя.
// Зависит только от портов, объявленных в domain.
package service

import (
	"context"
	"net/url"
	"strings"

	"github.com/Shipovmax/Lumora/internal/source/domain"
)

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) AddSource(ctx context.Context, userID string, typ domain.Type, name, url string) (domain.Source, error) {
	if !typ.Valid() {
		return domain.Source{}, domain.ErrInvalidType
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Source{}, domain.ErrNameRequired
	}

	rawURL := strings.TrimSpace(url)
	if rawURL == "" {
		return domain.Source{}, domain.ErrURLRequired
	}
	if err := validateURLScheme(rawURL); err != nil {
		return domain.Source{}, err
	}

	return s.repo.CreateSource(ctx, userID, typ, name, rawURL)
}

// validateURLScheme rejects schemes other than http/https at source-creation
// time (e.g. "file://"), so the error surfaces immediately as a 400 to the
// user instead of an opaque failure the next time ingest:fetch runs. This is
// a format check only — it does not (and cannot) rule out the URL resolving
// to a private/internal address at fetch time, since DNS can change or a
// redirect can point anywhere; that check happens where the actual outbound
// request is made (internal/source/fetcher, via a dial-time IP allowlist),
// not here.
func validateURLScheme(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return domain.ErrUnsupportedURLScheme
	}
	return nil
}

func (s *Service) ListSources(ctx context.Context, userID string) ([]domain.Source, error) {
	return s.repo.ListSources(ctx, userID)
}

func (s *Service) SetEnabled(ctx context.Context, userID, id string, enabled bool) (domain.Source, error) {
	return s.repo.SetEnabled(ctx, userID, id, enabled)
}

func (s *Service) DeleteSource(ctx context.Context, userID, id string) error {
	return s.repo.DeleteSource(ctx, userID, id)
}
