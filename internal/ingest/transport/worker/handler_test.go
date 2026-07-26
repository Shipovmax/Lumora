package ingestworker_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"

	ingestdomain "github.com/Shipovmax/Lumora/internal/ingest/domain"
	"github.com/Shipovmax/Lumora/internal/ingest/service"
	ingestworker "github.com/Shipovmax/Lumora/internal/ingest/transport/worker"
	"github.com/Shipovmax/Lumora/internal/platform/queue"
	sourcedomain "github.com/Shipovmax/Lumora/internal/source/domain"
)

type fakeSourceRepository struct {
	sources map[string]sourcedomain.Source
}

func (f *fakeSourceRepository) GetSourceByID(_ context.Context, id string) (sourcedomain.Source, error) {
	s, ok := f.sources[id]
	if !ok {
		return sourcedomain.Source{}, sourcedomain.ErrSourceNotFound
	}
	return s, nil
}

type fakeFetcher struct {
	posts []sourcedomain.RawPost
}

func (f *fakeFetcher) Fetch(_ context.Context, _ sourcedomain.Source) ([]sourcedomain.RawPost, error) {
	return f.posts, nil
}

type fakeRegistry struct {
	fetcher sourcedomain.Fetcher
}

func (f *fakeRegistry) For(_ sourcedomain.Type) (sourcedomain.Fetcher, error) {
	return f.fetcher, nil
}

type fakePostRepository struct{}

func (f *fakePostRepository) SaveNewPosts(_ context.Context, posts []ingestdomain.Post) ([]ingestdomain.Post, error) {
	for i := range posts {
		posts[i].ID = "post-" + posts[i].ExternalID
	}
	return posts, nil
}

type fakeEnqueuer struct {
	tasks []*asynq.Task
	err   error
}

func (f *fakeEnqueuer) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.tasks = append(f.tasks, task)
	return &asynq.TaskInfo{}, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTask(t *testing.T, payload queue.IngestFetchPayload) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return asynq.NewTask(queue.TypeIngestFetch, b)
}

func TestHandleFetchRejectsInvalidPayload(t *testing.T) {
	h := ingestworker.NewHandler(nil, &fakeEnqueuer{}, noopLogger())

	err := h.HandleFetch(context.Background(), asynq.NewTask(queue.TypeIngestFetch, []byte("not json")))
	require.Error(t, err)
}

func TestHandleFetchEnqueuesPipelineProcessForNewPosts(t *testing.T) {
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: true}
	svc := service.New(&fakePostRepository{},
		&fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}},
		&fakeRegistry{fetcher: &fakeFetcher{posts: []sourcedomain.RawPost{{ExternalID: "1"}}}},
	)
	enqueuer := &fakeEnqueuer{}
	h := ingestworker.NewHandler(svc, enqueuer, noopLogger())

	err := h.HandleFetch(context.Background(), newTask(t, queue.IngestFetchPayload{SourceID: "src-1"}))
	require.NoError(t, err)

	require.Len(t, enqueuer.tasks, 1)
	require.Equal(t, queue.TypePipelineProcess, enqueuer.tasks[0].Type())

	var payload queue.PipelineProcessPayload
	require.NoError(t, json.Unmarshal(enqueuer.tasks[0].Payload(), &payload))
	require.Equal(t, []string{"post-1"}, payload.PostIDs)
}

func TestHandleFetchDoesNotEnqueueWhenNoNewPosts(t *testing.T) {
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: true}
	svc := service.New(&fakePostRepository{},
		&fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}},
		&fakeRegistry{fetcher: &fakeFetcher{posts: nil}},
	)
	enqueuer := &fakeEnqueuer{}
	h := ingestworker.NewHandler(svc, enqueuer, noopLogger())

	err := h.HandleFetch(context.Background(), newTask(t, queue.IngestFetchPayload{SourceID: "src-1"}))
	require.NoError(t, err)
	require.Empty(t, enqueuer.tasks)
}

func TestHandleFetchPropagatesServiceError(t *testing.T) {
	svc := service.New(&fakePostRepository{}, &fakeSourceRepository{sources: map[string]sourcedomain.Source{}}, &fakeRegistry{})
	h := ingestworker.NewHandler(svc, &fakeEnqueuer{}, noopLogger())

	err := h.HandleFetch(context.Background(), newTask(t, queue.IngestFetchPayload{SourceID: "missing"}))
	require.ErrorIs(t, err, sourcedomain.ErrSourceNotFound)
}

func TestHandleFetchSucceedsEvenIfEnqueueFails(t *testing.T) {
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: true}
	svc := service.New(&fakePostRepository{},
		&fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}},
		&fakeRegistry{fetcher: &fakeFetcher{posts: []sourcedomain.RawPost{{ExternalID: "1"}}}},
	)
	enqueuer := &fakeEnqueuer{err: errors.New("redis down")}
	h := ingestworker.NewHandler(svc, enqueuer, noopLogger())

	err := h.HandleFetch(context.Background(), newTask(t, queue.IngestFetchPayload{SourceID: "src-1"}))
	require.NoError(t, err, "import already saved to DB, pipeline:process can be enqueued manually later")
}
