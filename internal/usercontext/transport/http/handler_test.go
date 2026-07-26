package usercontexthttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/usercontext/domain"
	"github.com/Shipovmax/Lumora/internal/usercontext/service"
	usercontexthttp "github.com/Shipovmax/Lumora/internal/usercontext/transport/http"
)

type fakeRepository struct {
	contexts map[string]domain.Context
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{contexts: map[string]domain.Context{}}
}

func (f *fakeRepository) GetContext(_ context.Context, userID string) (domain.Context, error) {
	if c, ok := f.contexts[userID]; ok {
		return c, nil
	}
	c := domain.Context{UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.contexts[userID] = c
	return c, nil
}

func (f *fakeRepository) UpdateContext(_ context.Context, userID, content string) (domain.Context, error) {
	c := domain.Context{UserID: userID, Content: content, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.contexts[userID] = c
	return c, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T) (*httptest.Server, *jwtauth.Issuer) {
	t.Helper()

	issuer := jwtauth.NewIssuer("test-secret", time.Hour)
	svc := service.New(newFakeRepository())
	handler := usercontexthttp.NewHandler(svc, noopLogger())

	r := chi.NewRouter()
	usercontexthttp.RegisterRoutes(r, handler, issuer.Middleware)

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

func TestGetContextRequiresBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, http.MethodGet, srv.URL+"/context", "", nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetContextCreatesEmptyContextOnFirstAccess(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodGet, srv.URL+"/context", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "user-1", body["user_id"])
	require.Equal(t, "", body["content"])
}

func TestUpdateContextReplacesContent(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPut, srv.URL+"/context", token, map[string]string{
		"content": "Interested in deep tech and AI research.",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "Interested in deep tech and AI research.", body["content"])
}

func TestUpdateContextRejectsContentTooLong(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPut, srv.URL+"/context", token, map[string]string{
		"content": strings.Repeat("a", 4001),
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrContextTooLong.Error(), body["error"])
}

func TestUpdateContextRejectsInvalidBody(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/context", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
