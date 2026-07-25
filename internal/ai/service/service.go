// Package service реализует бизнес-логику домена ai: генерацию и сохранение
// персонализированного объяснения события для конкретного пользователя.
// Зависит только от портов, объявленных в domain, и от узких интерфейсов
// (EventRepository, UserContextRepository), которые сам объявляет для нужных
// ему операций доменов pipeline и usercontext — без импорта их service/repository.
package service

import (
	"context"
	"fmt"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	pipelinedomain "github.com/Shipovmax/Lumora/internal/pipeline/domain"
	usercontextdomain "github.com/Shipovmax/Lumora/internal/usercontext/domain"
)

// EventRepository — часть domain.Repository пайплайна, нужная ai: чтение
// события по ID.
type EventRepository interface {
	GetEventByID(ctx context.Context, id string) (pipelinedomain.Event, error)
}

// UserContextRepository — часть domain.Repository usercontext, нужная ai:
// чтение AI-контекста пользователя.
type UserContextRepository interface {
	GetContext(ctx context.Context, userID string) (usercontextdomain.Context, error)
}

type Service struct {
	explanations aidomain.Repository
	events       EventRepository
	userContexts UserContextRepository
	provider     aidomain.Provider
}

func New(explanations aidomain.Repository, events EventRepository, userContexts UserContextRepository, provider aidomain.Provider) *Service {
	return &Service{explanations: explanations, events: events, userContexts: userContexts, provider: provider}
}

// GenerateExplanation генерирует и сохраняет персонализированное объяснение
// события для конкретного пользователя. Какие события релевантны какому
// пользователю (кого касается это событие) — решает Этап 9 (брифинг), не этот
// метод: здесь только механизм генерации для уже выбранной пары (event, user).
func (s *Service) GenerateExplanation(ctx context.Context, eventID, userID string) (aidomain.Explanation, error) {
	event, err := s.events.GetEventByID(ctx, eventID)
	if err != nil {
		return aidomain.Explanation{}, err
	}

	userCtx, err := s.userContexts.GetContext(ctx, userID)
	if err != nil {
		return aidomain.Explanation{}, err
	}

	result, err := s.provider.Explain(ctx, aidomain.EventInput{
		Title:   event.Title,
		Topic:   string(event.Topic),
		Content: event.MatchText,
	}, userCtx.Content)
	if err != nil {
		return aidomain.Explanation{}, fmt.Errorf("generate explanation for event %s: %w", eventID, err)
	}

	return s.explanations.SaveExplanation(ctx, aidomain.Explanation{
		EventID:            eventID,
		UserID:             userID,
		WhatHappened:       result.WhatHappened,
		WhyItHappened:      result.WhyItHappened,
		WhatChanged:        result.WhatChanged,
		WhatItMeansForUser: result.WhatItMeansForUser,
		Model:              result.Model,
	})
}
