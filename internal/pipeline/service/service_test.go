package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
	"github.com/Shipovmax/Lumora/internal/pipeline/service"
)

// fakeRepository — in-memory реализация domain.Repository для unit-тестов сервиса.
type fakeRepository struct {
	posts  map[string]domain.PostRef
	events map[string]domain.Event
	nextID int
}

func newFakeRepository(posts []domain.PostRef) *fakeRepository {
	r := &fakeRepository{posts: map[string]domain.PostRef{}, events: map[string]domain.Event{}}
	for _, p := range posts {
		r.posts[p.ID] = p
	}
	return r
}

func (r *fakeRepository) genID() string {
	r.nextID++
	return fmt.Sprintf("event-%d", r.nextID)
}

func (r *fakeRepository) GetPosts(_ context.Context, postIDs []string) ([]domain.PostRef, error) {
	out := make([]domain.PostRef, 0, len(postIDs))
	for _, id := range postIDs {
		if p, ok := r.posts[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeRepository) ListRecentEvents(_ context.Context, since time.Time) ([]domain.Event, error) {
	var out []domain.Event
	for _, e := range r.events {
		if !e.LastSeenAt.Before(since) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (r *fakeRepository) GetEventByID(_ context.Context, id string) (domain.Event, error) {
	e, ok := r.events[id]
	if !ok {
		return domain.Event{}, domain.ErrEventNotFound
	}
	return e, nil
}

func (r *fakeRepository) CreateEventWithPost(_ context.Context, topic domain.Topic, title, matchText, postID string, publishedAt time.Time) (domain.Event, error) {
	e := domain.Event{
		ID:          r.genID(),
		Topic:       topic,
		Title:       title,
		MatchText:   matchText,
		Importance:  20,
		SourceCount: 1,
		FirstSeenAt: publishedAt,
		LastSeenAt:  publishedAt,
	}
	r.events[e.ID] = e
	return e, nil
}

func (r *fakeRepository) AttachPost(_ context.Context, eventID, postID, matchText string, publishedAt time.Time) (domain.Event, error) {
	e, ok := r.events[eventID]
	if !ok {
		return domain.Event{}, fmt.Errorf("unknown event %q", eventID)
	}
	e.MatchText = matchText
	e.SourceCount++
	e.Importance += 20
	if publishedAt.After(e.LastSeenAt) {
		e.LastSeenAt = publishedAt
	}
	r.events[eventID] = e
	return e, nil
}

func TestProcessCreatesNewEventWhenNoMatch(t *testing.T) {
	ctx := context.Background()
	posts := []domain.PostRef{
		{ID: "p1", SourceID: "s1", Title: "OpenAI releases new model", Content: "Anthropic and OpenAI compete on AI safety", PublishedAt: time.Now()},
	}
	repo := newFakeRepository(posts)
	svc := service.New(repo)

	events, err := svc.Process(ctx, []string{"p1"})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, domain.TopicAI, events[0].Topic)
	require.Equal(t, 1, events[0].SourceCount)
}

func TestProcessAttachesSimilarPostToExistingEvent(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	p1 := domain.PostRef{ID: "p1", SourceID: "s1", Title: "Bitcoin surges past new high", Content: "Bitcoin price rallies amid ETF inflows", PublishedAt: now}
	p2 := domain.PostRef{ID: "p2", SourceID: "s2", Title: "Bitcoin price rallies past record high", Content: "Bitcoin surges amid strong ETF inflows this week", PublishedAt: now.Add(time.Minute)}

	repo := newFakeRepository([]domain.PostRef{p1, p2})
	svc := service.New(repo)

	firstBatch, err := svc.Process(ctx, []string{"p1"})
	require.NoError(t, err)
	require.Len(t, firstBatch, 1)
	eventID := firstBatch[0].ID

	secondBatch, err := svc.Process(ctx, []string{"p2"})
	require.NoError(t, err)
	require.Len(t, secondBatch, 1)
	require.Equal(t, eventID, secondBatch[0].ID, "similar post from a different source should join the existing event")
	require.Equal(t, 2, secondBatch[0].SourceCount)
}

func TestProcessClustersSimilarPostsWithinSameBatch(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	p1 := domain.PostRef{ID: "p1", Title: "Bitcoin surges past new high", Content: "Bitcoin price rallies amid ETF inflows", PublishedAt: now}
	p2 := domain.PostRef{ID: "p2", Title: "Bitcoin price rallies past record high", Content: "Bitcoin surges amid strong ETF inflows this week", PublishedAt: now.Add(time.Minute)}

	repo := newFakeRepository([]domain.PostRef{p1, p2})
	svc := service.New(repo)

	events, err := svc.Process(ctx, []string{"p1", "p2"})
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, events[0].ID, events[1].ID, "similar posts in the same batch should cluster into one event")
}

func TestProcessIgnoresEventsOutsideClusterWindow(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	p1 := domain.PostRef{ID: "p1", Title: "Bitcoin surges past new high", Content: "Bitcoin price rallies amid ETF inflows", PublishedAt: now.Add(-72 * time.Hour)}
	p2 := domain.PostRef{ID: "p2", Title: "Bitcoin price rallies past record high", Content: "Bitcoin surges amid strong ETF inflows this week", PublishedAt: now}

	repo := newFakeRepository([]domain.PostRef{p1, p2})
	svc := service.New(repo)

	first, err := svc.Process(ctx, []string{"p1"})
	require.NoError(t, err)

	// Искусственно «состариваем» событие, чтобы оно вышло за clusterWindow.
	e := repo.events[first[0].ID]
	e.LastSeenAt = now.Add(-72 * time.Hour)
	repo.events[first[0].ID] = e

	second, err := svc.Process(ctx, []string{"p2"})
	require.NoError(t, err)
	require.NotEqual(t, first[0].ID, second[0].ID, "event outside the cluster window must not be reused")
}

func TestProcessSkipsUnknownPostIDs(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepository(nil)
	svc := service.New(repo)

	events, err := svc.Process(ctx, []string{"missing"})
	require.NoError(t, err)
	require.Empty(t, events)
}
