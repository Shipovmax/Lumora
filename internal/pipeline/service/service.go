// Package service реализует бизнес-логику домена pipeline: кластеризацию
// импортированных публикаций в события. Зависит только от портов,
// объявленных в domain.
package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
)

const (
	// clusterWindow — окно, в котором событие считается ещё активным и может
	// принять новую публикацию; более старые события кандидатами не рассматриваются.
	clusterWindow = 48 * time.Hour
	// similarityThreshold — порог Jaccard-сходства токенов заголовка+текста, выше
	// которого публикация присоединяется к событию, а не образует новое.
	// Подобрано как MVP-эвристика, порог — кандидат на пересмотр по метрикам.
	similarityThreshold = 0.3
	// maxMatchTextLength ограничивает рост match_text по мере присоединения
	// новых публикаций к событию (сравнение остаётся быстрым на масштабе).
	maxMatchTextLength = 4000
)

type Service struct {
	repo domain.Repository
	now  func() time.Time
}

func New(repo domain.Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// clusterCandidate — событие-кандидат для присоединения новой публикации:
// либо уже существующее (последние clusterWindow), либо только что созданное
// в рамках текущего батча (чтобы несколько публикаций одного события,
// пришедших в одном вызове Process, тоже склеились друг с другом).
type clusterCandidate struct {
	eventID   string
	tokens    map[string]struct{}
	matchText string
}

// Process кластеризует публикации в события: для каждой публикации (в порядке
// времени публикации) ищет наиболее похожее недавнее/только что созданное
// событие и присоединяет к нему при сходстве выше порога, иначе создаёт новое.
func (s *Service) Process(ctx context.Context, postIDs []string) ([]domain.Event, error) {
	posts, err := s.repo.GetPosts(ctx, postIDs)
	if err != nil {
		return nil, err
	}
	if len(posts) == 0 {
		return nil, nil
	}

	sort.Slice(posts, func(i, j int) bool { return posts[i].PublishedAt.Before(posts[j].PublishedAt) })

	recentEvents, err := s.repo.ListRecentEvents(ctx, s.now().Add(-clusterWindow))
	if err != nil {
		return nil, err
	}

	candidates := make([]*clusterCandidate, 0, len(recentEvents))
	for _, e := range recentEvents {
		candidates = append(candidates, &clusterCandidate{
			eventID:   e.ID,
			tokens:    tokenize(e.MatchText),
			matchText: e.MatchText,
		})
	}

	results := make([]domain.Event, 0, len(posts))

	for _, post := range posts {
		publishedAt := post.PublishedAt
		if publishedAt.IsZero() {
			publishedAt = s.now()
		}

		tokens := tokenize(post.Title + " " + post.Content)
		best, bestScore := findBestMatch(candidates, tokens)

		if best != nil && bestScore >= similarityThreshold {
			mergedText := mergeMatchText(best.matchText, post.Title, post.Content)

			event, err := s.repo.AttachPost(ctx, best.eventID, post.ID, mergedText, publishedAt)
			if err != nil {
				return nil, err
			}

			best.matchText = mergedText
			best.tokens = tokenize(mergedText)

			results = append(results, event)
			continue
		}

		topic := classifyTopic(post.Title, post.Content)
		matchText := mergeMatchText("", post.Title, post.Content)

		event, err := s.repo.CreateEventWithPost(ctx, topic, post.Title, matchText, post.ID, publishedAt)
		if err != nil {
			return nil, err
		}

		candidates = append(candidates, &clusterCandidate{
			eventID:   event.ID,
			tokens:    tokenize(matchText),
			matchText: matchText,
		})

		results = append(results, event)
	}

	return results, nil
}

func findBestMatch(candidates []*clusterCandidate, tokens map[string]struct{}) (*clusterCandidate, float64) {
	var best *clusterCandidate
	bestScore := 0.0

	for _, c := range candidates {
		score := jaccardSimilarity(c.tokens, tokens)
		if score > bestScore {
			best = c
			bestScore = score
		}
	}

	return best, bestScore
}

func mergeMatchText(existing, title, content string) string {
	merged := strings.TrimSpace(strings.Join([]string{existing, title, content}, " "))
	if len(merged) > maxMatchTextLength {
		merged = merged[:maxMatchTextLength]
	}
	return merged
}
