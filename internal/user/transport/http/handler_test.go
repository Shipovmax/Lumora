package userhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/user/domain"
	"github.com/Shipovmax/Lumora/internal/user/service"
	userhttp "github.com/Shipovmax/Lumora/internal/user/transport/http"
)

// fakeRepository — in-memory реализация user/domain.Repository. GetProfile
// создаёт пустой профиль при первом обращении — тот же контракт, что и у
// настоящего Postgres-репозитория (см. ARCHITECTURE.md §7).
type fakeRepository struct {
	profiles map[string]domain.Profile
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{profiles: map[string]domain.Profile{}}
}

func (f *fakeRepository) GetProfile(_ context.Context, userID string) (domain.Profile, error) {
	if p, ok := f.profiles[userID]; ok {
		return p, nil
	}
	p := domain.Profile{UserID: userID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.profiles[userID] = p
	return p, nil
}

func (f *fakeRepository) UpdateProfile(_ context.Context, userID string, update domain.ProfileUpdate) (domain.Profile, error) {
	p := domain.Profile{
		UserID: userID, Name: update.Name, Country: update.Country, Language: update.Language,
		Profession: update.Profession, Interests: update.Interests, Topics: update.Topics,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	f.profiles[userID] = p
	return p, nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T) (*httptest.Server, *jwtauth.Issuer) {
	t.Helper()

	issuer := jwtauth.NewIssuer("test-secret", time.Hour)
	svc := service.New(newFakeRepository())
	handler := userhttp.NewHandler(svc, noopLogger())

	r := chi.NewRouter()
	userhttp.RegisterRoutes(r, handler, issuer.Middleware)

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

func TestGetProfileRequiresBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, http.MethodGet, srv.URL+"/user/profile", "", nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGetProfileCreatesEmptyProfileOnFirstAccess(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodGet, srv.URL+"/user/profile", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "user-1", body["user_id"])
	require.Equal(t, "", body["name"])
}

func TestUpdateProfileReplacesFields(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPut, srv.URL+"/user/profile", token, map[string]any{
		"name": "Ada Lovelace", "country": "UK", "language": "en",
		"profession": "Mathematician", "interests": []string{"computing"}, "topics": []string{"ai"},
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "Ada Lovelace", body["name"])
	require.Equal(t, []any{"computing"}, body["interests"])
}

func TestUpdateProfileRejectsNameTooLong(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	longName := ""
	for i := 0; i < 201; i++ {
		longName += "a"
	}

	resp := do(t, http.MethodPut, srv.URL+"/user/profile", token, map[string]any{"name": longName})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrNameTooLong.Error(), body["error"])
}

func TestUpdateProfileRejectsInvalidBody(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, srv.URL+"/user/profile", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
