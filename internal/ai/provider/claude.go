// Package provider содержит реализации domain.Provider по вендору AI.
// Выбор вендора (Anthropic Claude) согласован с пользователем 2026-07-25 —
// см. ARCHITECTURE.md, Этап 8; domain.Provider остаётся вендор-независимым.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/Shipovmax/Lumora/internal/ai/domain"
)

// model — фиксированная модель для генерации объяснений; см. skill claude-api:
// claude-opus-5 — модель по умолчанию, если пользователь явно не просит другую.
const model = "claude-opus-5"

const maxTokens = 4096

// ClaudeProvider реализует domain.Provider через Anthropic Messages API со
// структурированным JSON-выводом (output_config.format) — так же надёжно, как
// assistant prefill, но prefill не поддерживается на этой модели.
type ClaudeProvider struct {
	client anthropic.Client
}

// NewClaudeProvider создаёт клиента Anthropic. Ключ читается из
// ANTHROPIC_API_KEY (стандартное разрешение SDK — см. skill claude-api);
// отдельного поля в internal/config нет, так как ключ нужен только воркеру,
// а не cmd/api.
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{client: anthropic.NewClient()}
}

var _ domain.Provider = (*ClaudeProvider)(nil)

var explanationSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"what_happened":          map[string]any{"type": "string", "description": "Что произошло: краткое объективное описание события."},
		"why_it_happened":        map[string]any{"type": "string", "description": "Почему это произошло: причины и контекст события."},
		"what_changed":           map[string]any{"type": "string", "description": "Что изменилось в результате события."},
		"what_it_means_for_user": map[string]any{"type": "string", "description": "Что это значит лично для пользователя, с учётом его контекста."},
	},
	"required":             []string{"what_happened", "why_it_happened", "what_changed", "what_it_means_for_user"},
	"additionalProperties": false,
}

const systemPrompt = `Ты — модуль объяснения событий в Lumora, приложении персонализированных новостных брифингов.
По заголовку и содержанию события сформируй объяснение из четырёх частей на русском языке: что произошло, почему это произошло, что изменилось, и что это значит лично для пользователя.
Последнюю часть обязательно персонализируй с учётом контекста пользователя, если он предоставлен — свяжи событие с его интересами, профессией или темами, которые он отслеживает. Если контекст пуст, дай нейтральный, но содержательный ответ, не выдумывая деталей о пользователе.
Опирайся только на предоставленный текст события — не добавляй факты, которых там нет. Отвечай кратко: 1-3 предложения на каждую часть.`

type explanationJSON struct {
	WhatHappened       string `json:"what_happened"`
	WhyItHappened      string `json:"why_it_happened"`
	WhatChanged        string `json:"what_changed"`
	WhatItMeansForUser string `json:"what_it_means_for_user"`
}

func (p *ClaudeProvider) Explain(ctx context.Context, event domain.EventInput, userContext string) (domain.ProviderResult, error) {
	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffortMedium,
			Format: anthropic.JSONOutputFormatParam{Schema: explanationSchema},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(buildPrompt(event, userContext))),
		},
	})
	if err != nil {
		return domain.ProviderResult{}, fmt.Errorf("claude: generate explanation: %w", err)
	}

	if resp.StopReason == anthropic.StopReasonRefusal {
		return domain.ProviderResult{}, fmt.Errorf("claude: refused to generate explanation (stop_details=%v)", resp.StopDetails)
	}

	for _, block := range resp.Content {
		text, ok := block.AsAny().(anthropic.TextBlock)
		if !ok {
			continue
		}

		var parsed explanationJSON
		if err := json.Unmarshal([]byte(text.Text), &parsed); err != nil {
			return domain.ProviderResult{}, fmt.Errorf("claude: parse explanation: %w", err)
		}

		return domain.ProviderResult{
			WhatHappened:       parsed.WhatHappened,
			WhyItHappened:      parsed.WhyItHappened,
			WhatChanged:        parsed.WhatChanged,
			WhatItMeansForUser: parsed.WhatItMeansForUser,
			Model:              string(resp.Model),
		}, nil
	}

	return domain.ProviderResult{}, fmt.Errorf("claude: no text content in response (stop_reason=%s)", resp.StopReason)
}

func buildPrompt(event domain.EventInput, userContext string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Тема: %s\nЗаголовок: %s\nСодержание: %s\n", event.Topic, event.Title, event.Content)

	if userContext == "" {
		b.WriteString("\nКонтекст пользователя: не указан.\n")
	} else {
		fmt.Fprintf(&b, "\nКонтекст пользователя: %s\n", userContext)
	}

	return b.String()
}
