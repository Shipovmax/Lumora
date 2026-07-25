package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	ingestdomain "github.com/Shipovmax/Lumora/internal/ingest/domain"
	"github.com/Shipovmax/Lumora/internal/ingest/service"
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
	err   error
}

func (f *fakeFetcher) Fetch(_ context.Context, _ sourcedomain.Source) ([]sourcedomain.RawPost, error) {
	return f.posts, f.err
}

type fakeRegistry struct {
	fetcher sourcedomain.Fetcher
}

func (f *fakeRegistry) For(_ sourcedomain.Type) (sourcedomain.Fetcher, error) {
	return f.fetcher, nil
}

type fakePostRepository struct {
	saved []ingestdomain.Post
}

func (f *fakePostRepository) SaveNewPosts(_ context.Context, posts []ingestdomain.Post) ([]ingestdomain.Post, error) {
	f.saved = append(f.saved, posts...)
	return posts, nil
}

func TestImportSourceCleansTextAndSaves(t *testing.T) {
	ctx := context.Background()
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: true}

	fetcher := &fakeFetcher{posts: []sourcedomain.RawPost{
		{ExternalID: "1", Title: "<b>Hello</b>", Content: "Line one\n\n  Line   two &amp; more"},
	}}
	posts := &fakePostRepository{}

	svc := service.New(posts, &fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}}, &fakeRegistry{fetcher: fetcher})

	saved, err := svc.ImportSource(ctx, "src-1")
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, "src-1", saved[0].SourceID)
	require.Equal(t, "Hello", saved[0].Title)
	require.Equal(t, "Line one Line two & more", saved[0].Content)
}

func TestImportSourceSkipsDisabledSource(t *testing.T) {
	ctx := context.Background()
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: false}

	fetcher := &fakeFetcher{posts: []sourcedomain.RawPost{{ExternalID: "1"}}}
	posts := &fakePostRepository{}

	svc := service.New(posts, &fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}}, &fakeRegistry{fetcher: fetcher})

	saved, err := svc.ImportSource(ctx, "src-1")
	require.NoError(t, err)
	require.Empty(t, saved)
	require.Empty(t, posts.saved)
}

func TestImportSourceSkipsPostsWithoutExternalID(t *testing.T) {
	ctx := context.Background()
	src := sourcedomain.Source{ID: "src-1", Type: sourcedomain.TypeRSS, Enabled: true}

	fetcher := &fakeFetcher{posts: []sourcedomain.RawPost{
		{ExternalID: "", Title: "No id"},
		{ExternalID: "2", Title: "Has id"},
	}}
	posts := &fakePostRepository{}

	svc := service.New(posts, &fakeSourceRepository{sources: map[string]sourcedomain.Source{"src-1": src}}, &fakeRegistry{fetcher: fetcher})

	saved, err := svc.ImportSource(ctx, "src-1")
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, "2", saved[0].ExternalID)
}

func TestImportSourceUnknownSource(t *testing.T) {
	ctx := context.Background()
	svc := service.New(&fakePostRepository{}, &fakeSourceRepository{sources: map[string]sourcedomain.Source{}}, &fakeRegistry{})

	_, err := svc.ImportSource(ctx, "missing")
	require.ErrorIs(t, err, sourcedomain.ErrSourceNotFound)
}
