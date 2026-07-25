package domain

import "context"

// EventInput — данные о событии, нужные Provider для генерации объяснения:
// минимальный срез, не сама pipeline.domain.Event (никаких прямых
// междоменных импортов бизнес-логики — см. остальные домены).
type EventInput struct {
	Title   string
	Topic   string
	Content string
}

// ProviderResult — четыре блока, сгенерированные Provider. Не содержит
// EventID/UserID/Model — их добавляет service перед сохранением.
type ProviderResult struct {
	WhatHappened       string
	WhyItHappened      string
	WhatChanged        string
	WhatItMeansForUser string
	// Model — идентификатор модели, фактически обработавшей запрос
	// (для аудита; провайдер сам решает, что туда положить).
	Model string
}

// Provider — порт генерации объяснения события AI-моделью с учётом
// пользовательского контекста. Реализация — internal/ai/provider
// (Anthropic Claude, согласовано с пользователем 2026-07-25); интерфейс не
// завязан на конкретного вендора.
type Provider interface {
	Explain(ctx context.Context, event EventInput, userContext string) (ProviderResult, error)
}
