// Package service реализует бизнес-логику домена user: чтение и редактирование
// профиля пользователя. Зависит только от портов, объявленных в domain.
package service

import (
	"context"
	"strings"

	"github.com/Shipovmax/Lumora/internal/user/domain"
)

const maxNameLength = 200

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, userID string) (domain.Profile, error) {
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, update domain.ProfileUpdate) (domain.Profile, error) {
	normalized, err := normalizeUpdate(update)
	if err != nil {
		return domain.Profile{}, err
	}
	return s.repo.UpdateProfile(ctx, userID, normalized)
}

func normalizeUpdate(update domain.ProfileUpdate) (domain.ProfileUpdate, error) {
	name := strings.TrimSpace(update.Name)
	if len(name) > maxNameLength {
		return domain.ProfileUpdate{}, domain.ErrNameTooLong
	}

	return domain.ProfileUpdate{
		Name:       name,
		Country:    strings.TrimSpace(update.Country),
		Language:   strings.TrimSpace(update.Language),
		Profession: strings.TrimSpace(update.Profession),
		Interests:  normalizeList(update.Interests),
		Topics:     normalizeList(update.Topics),
	}, nil
}

// normalizeList обрезает пробелы и отбрасывает пустые элементы, возвращая непустой
// (не nil) срез, чтобы sqlc/pgx корректно записывали TEXT[] NOT NULL DEFAULT '{}'.
func normalizeList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
