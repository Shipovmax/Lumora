// Package service реализует бизнес-логику домена briefing: отбор релевантных
// пользователю событий, получение/генерация их объяснений (Этап 8) и сборку
// брифинга. Зависит только от портов, объявленных в domain, и от узких
// интерфейсов (ExplanationRepository, ExplanationGenerator), которые сам
// объявляет для нужных ему операций домена ai — без импорта его service/repository.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	aidomain "github.com/Shipovmax/Lumora/internal/ai/domain"
	"github.com/Shipovmax/Lumora/internal/briefing/domain"
)

const (
	// lookbackWindow — окно с предыдущего цикла (утро/вечер, см. cron-расписание
	// в cmd/worker): 12 часов между запусками планировщика.
	lookbackWindow = 12 * time.Hour
	// maxEventsPerBriefing ограничивает объём брифинга наиболее важными событиями.
	maxEventsPerBriefing = 10
)

// ExplanationRepository — часть domain.Repository ai, нужная брифингу: чтение
// уже сгенерированного объяснения (чтобы не дублировать AI-вызов).
type ExplanationRepository interface {
	GetExplanation(ctx context.Context, eventID, userID string) (aidomain.Explanation, error)
}

// ExplanationGenerator — часть service ai, нужная брифингу: генерация
// объяснения для пары (событие, пользователь), если его ещё нет.
type ExplanationGenerator interface {
	GenerateExplanation(ctx context.Context, eventID, userID string) (aidomain.Explanation, error)
}

type Service struct {
	repo         domain.Repository
	explanations ExplanationRepository
	generator    ExplanationGenerator
	log          *slog.Logger
	now          func() time.Time
}

func New(repo domain.Repository, explanations ExplanationRepository, generator ExplanationGenerator, log *slog.Logger) *Service {
	return &Service{repo: repo, explanations: explanations, generator: generator, log: log, now: time.Now}
}

// Build формирует брифинг для пользователя: отбирает релевантные события по
// его источникам (ещё не включённые в предыдущие брифинги пользователя), для
// каждого получает или генерирует персонализированное объяснение и сохраняет
// брифинг. ErrNoRelevantEvents — не ошибка обработки, а нормальное состояние
// (слать нечего): вызывающий код (asynq-обработчик) не должен считать это сбоем.
func (s *Service) Build(ctx context.Context, userID string, typ domain.Type) (domain.Briefing, error) {
	if !typ.Valid() {
		return domain.Briefing{}, domain.ErrInvalidType
	}

	since := s.now().Add(-lookbackWindow)

	candidates, err := s.repo.ListCandidateEvents(ctx, userID, since, maxEventsPerBriefing)
	if err != nil {
		return domain.Briefing{}, err
	}
	if len(candidates) == 0 {
		return domain.Briefing{}, domain.ErrNoRelevantEvents
	}

	events := make([]domain.BriefingEvent, 0, len(candidates))
	for _, c := range candidates {
		exp, err := s.explanationFor(ctx, c.ID, userID)
		if err != nil {
			s.log.Error("briefing: skip event without explanation",
				slog.String("event_id", c.ID), slog.Any("error", err))
			continue
		}

		events = append(events, domain.BriefingEvent{
			EventID:            c.ID,
			Topic:              c.Topic,
			Title:              c.Title,
			Importance:         c.Importance,
			Rank:               len(events) + 1,
			WhatHappened:       exp.WhatHappened,
			WhyItHappened:      exp.WhyItHappened,
			WhatChanged:        exp.WhatChanged,
			WhatItMeansForUser: exp.WhatItMeansForUser,
		})
	}

	if len(events) == 0 {
		return domain.Briefing{}, domain.ErrNoRelevantEvents
	}

	eventIDs := make([]string, len(events))
	for i, e := range events {
		eventIDs[i] = e.EventID
	}

	id, generatedAt, err := s.repo.CreateBriefing(ctx, userID, typ, eventIDs)
	if err != nil {
		return domain.Briefing{}, fmt.Errorf("save briefing: %w", err)
	}

	return domain.Briefing{
		ID:          id,
		UserID:      userID,
		Type:        typ,
		GeneratedAt: generatedAt,
		Events:      events,
	}, nil
}

func (s *Service) explanationFor(ctx context.Context, eventID, userID string) (aidomain.Explanation, error) {
	exp, err := s.explanations.GetExplanation(ctx, eventID, userID)
	if err == nil {
		return exp, nil
	}
	if !errors.Is(err, aidomain.ErrExplanationNotFound) {
		return aidomain.Explanation{}, err
	}
	return s.generator.GenerateExplanation(ctx, eventID, userID)
}
