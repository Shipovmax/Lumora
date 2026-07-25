package service

import (
	"regexp"
	"strings"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
)

// topicOrder задаёт приоритет при пересечении ключевых слов нескольких тем.
var topicOrder = []domain.Topic{domain.TopicAI, domain.TopicCrypto, domain.TopicEconomy, domain.TopicWorld}

// topicKeywords — MVP-эвристика классификации темы события: первое совпадение
// ключевого слова по фиксированному списку (см. README, пример брифинга).
// Кандидат на замену AI-классификацией позже — не входит в объём Этапа 7.
var topicKeywords = map[domain.Topic][]string{
	domain.TopicAI: {
		"ai", "artificial intelligence", "llm", "gpt", "openai", "anthropic",
		"claude", "machine learning", "neural network", "chatbot",
	},
	domain.TopicCrypto: {
		"crypto", "bitcoin", "btc", "ethereum", "eth", "blockchain", "token",
		"defi", "nft", "stablecoin",
	},
	domain.TopicEconomy: {
		"economy", "economic", "inflation", "gdp", "stock market", "stocks",
		"federal reserve", "interest rate", "recession", "unemployment",
	},
	domain.TopicWorld: {
		"war", "election", "president", "government", "conflict", "diplomacy",
		"united nations", "sanctions", "military", "protest",
	},
}

// topicPatterns компилирует ключевые слова в regexp с границами слов (\b), чтобы
// избежать ложных срабатываний по подстроке — например, "war" не должен матчить
// "award".
var topicPatterns = compileTopicPatterns()

func compileTopicPatterns() map[domain.Topic][]*regexp.Regexp {
	patterns := make(map[domain.Topic][]*regexp.Regexp, len(topicKeywords))
	for topic, keywords := range topicKeywords {
		compiled := make([]*regexp.Regexp, 0, len(keywords))
		for _, keyword := range keywords {
			compiled = append(compiled, regexp.MustCompile(`\b`+regexp.QuoteMeta(keyword)+`\b`))
		}
		patterns[topic] = compiled
	}
	return patterns
}

func classifyTopic(title, content string) domain.Topic {
	text := strings.ToLower(title + " " + content)
	for _, topic := range topicOrder {
		for _, pattern := range topicPatterns[topic] {
			if pattern.MatchString(text) {
				return topic
			}
		}
	}
	return domain.TopicOther
}
