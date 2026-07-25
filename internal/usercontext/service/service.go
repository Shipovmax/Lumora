// Package service реализует бизнес-логику домена usercontext: чтение и
// редактирование AI-контекста пользователя. Зависит только от портов,
// объявленных в domain.
package service

import (
	"context"
	"strings"

	"github.com/Shipovmax/Lumora/internal/usercontext/domain"
)

const maxContentLength = 4000

type Service struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetContext(ctx context.Context, userID string) (domain.Context, error) {
	return s.repo.GetContext(ctx, userID)
}

func (s *Service) UpdateContext(ctx context.Context, userID, content string) (domain.Context, error) {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) > maxContentLength {
		return domain.Context{}, domain.ErrContextTooLong
	}
	return s.repo.UpdateContext(ctx, userID, trimmed)
}
