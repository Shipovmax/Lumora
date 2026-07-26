package sourcehttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/source/domain"
	"github.com/Shipovmax/Lumora/internal/source/service"
	sourcehttp "github.com/Shipovmax/Lumora/internal/source/transport/http"
)

// fakeRepository — in-memory реализация source/domain.Repository, ownership
// (userID) проверяется так же, как в реальном Postgres-репозитории: чужой
// источник не виден и не редактируется, а не проваливается с internal error.
type fakeRepository struct {
	mu      sync.Mutex
	sources map[string]domain.Source
	nextID  int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{sources: map[string]domain.Source{}}
}

func (f *fakeRepository) genID() string {
	f.nextID++
	return fmt.Sprintf("source-%d", f.nextID)
}

func (f *fakeRepository) CreateSource(_ context.Context, userID string, typ domain.Type, name, url string) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s := domain.Source{
		ID: f.genID(), UserID: userID, Type: typ, Name: name, URL: url, Enabled: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.sources[s.ID] = s
	return s, nil
}

func (f *fakeRepository) ListSources(_ context.Context, userID string) ([]domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []domain.Source
	for _, s := range f.sources {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeRepository) GetSource(_ context.Context, userID, id string) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.Source{}, domain.ErrSourceNotFound
	}
	return s, nil
}

func (f *fakeRepository) GetSourceByID(_ context.Context, id string) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok {
		return domain.Source{}, domain.ErrSourceNotFound
	}
	return s, nil
}

func (f *fakeRepository) SetEnabled(_ context.Context, userID, id string, enabled bool) (domain.Source, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.Source{}, domain.ErrSourceNotFound
	}
	s.Enabled = enabled
	s.UpdatedAt = time.Now()
	f.sources[id] = s
	return s, nil
}

func (f *fakeRepository) DeleteSource(_ context.Context, userID, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, ok := f.sources[id]
	if !ok || s.UserID != userID {
		return domain.ErrSourceNotFound
	}
	delete(f.sources, id)
	return nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T) (*httptest.Server, *jwtauth.Issuer) {
	t.Helper()

	issuer := jwtauth.NewIssuer("test-secret", time.Hour)
	svc := service.New(newFakeRepository())
	handler := sourcehttp.NewHandler(svc, noopLogger())

	r := chi.NewRouter()
	sourcehttp.RegisterRoutes(r, handler, issuer.Middleware)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, issuer
}

func do(t *testing.T, method, url, bearer string, body any) *http.Response {
	t.Helper()

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func decodeList(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	var body []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

func TestCreateSourceRequiresBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", "", map[string]string{"type": "rss", "name": "HN", "url": "https://x"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCreateSourceSuccess(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", token, map[string]string{
		"type": "rss", "name": "Hacker News", "url": "https://news.ycombinator.com/rss",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "rss", body["type"])
	require.Equal(t, true, body["enabled"])
}

func TestCreateSourceRejectsInvalidType(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", token, map[string]string{
		"type": "carrier-pigeon", "name": "N", "url": "https://x",
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrInvalidType.Error(), body["error"])
}

func TestCreateSourceRejectsUnsupportedURLScheme(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", token, map[string]string{
		"type": "rss", "name": "N", "url": "file:///etc/passwd",
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrUnsupportedURLScheme.Error(), body["error"])
}

func TestListSourcesOnlyReturnsCallersOwnSources(t *testing.T) {
	srv, issuer := newTestServer(t)
	tokenA, _, err := issuer.IssueAccessToken("user-a")
	require.NoError(t, err)
	tokenB, _, err := issuer.IssueAccessToken("user-b")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", tokenA, map[string]string{"type": "rss", "name": "A", "url": "https://a"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp = do(t, http.MethodGet, srv.URL+"/sources/", tokenB, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Empty(t, decodeList(t, resp))

	resp = do(t, http.MethodGet, srv.URL+"/sources/", tokenA, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, decodeList(t, resp), 1)
}

func TestSetEnabledRejectsSourceOwnedByAnotherUser(t *testing.T) {
	srv, issuer := newTestServer(t)
	tokenA, _, err := issuer.IssueAccessToken("user-a")
	require.NoError(t, err)
	tokenB, _, err := issuer.IssueAccessToken("user-b")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", tokenA, map[string]string{"type": "rss", "name": "A", "url": "https://a"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	created := decode(t, resp)
	id := created["id"].(string)

	resp = do(t, http.MethodPatch, srv.URL+"/sources/"+id, tokenB, map[string]bool{"enabled": false})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrSourceNotFound.Error(), body["error"])
}

func TestDeleteSourceRemovesIt(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/sources/", token, map[string]string{"type": "rss", "name": "A", "url": "https://a"})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	created := decode(t, resp)
	id := created["id"].(string)

	resp = do(t, http.MethodDelete, srv.URL+"/sources/"+id, token, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp = do(t, http.MethodGet, srv.URL+"/sources/", token, nil)
	require.Empty(t, decodeList(t, resp))
}
