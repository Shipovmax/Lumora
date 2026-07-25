package service

import (
	"strings"
	"unicode"
)

// tokenize разбивает текст на набор значимых слов: нижний регистр, только
// буквенно-цифровые последовательности длиной от 3 символов (в байтах —
// приближение, но для эвристики сходства этого достаточно, стоп-слова и
// стемминг сознательно не делаем — MVP).
func tokenize(text string) map[string]struct{} {
	tokens := map[string]struct{}{}
	var b strings.Builder

	flush := func() {
		if b.Len() >= 3 {
			tokens[strings.ToLower(b.String())] = struct{}{}
		}
		b.Reset()
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	return tokens
}

// jaccardSimilarity — доля общих токенов от размера объединения множеств.
// Используется как эвристика близости для дедупликации/кластеризации публикаций
// в события (Этап 7); без embeddings/ML — простое и объяснимое решение для MVP.
func jaccardSimilarity(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	intersection := 0
	for token := range a {
		if _, ok := b[token]; ok {
			intersection++
		}
	}

	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
