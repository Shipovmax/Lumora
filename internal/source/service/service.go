// Package service реализует бизнес-логику домена source: добавление,
// просмотр, включение/выключение и удаление источников пользователя.
// Зависит только от портов, объявленных в domain.
package service

import (
	"context"
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

	url = strings.TrimSpace(url)
	if url == "" {
		return domain.Source{}, domain.ErrURLRequired
	}

	return s.repo.CreateSource(ctx, userID, typ, name, url)
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
