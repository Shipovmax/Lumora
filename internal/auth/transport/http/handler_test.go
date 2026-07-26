package authhttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
	"github.com/Shipovmax/Lumora/internal/auth/service"
	authhttp "github.com/Shipovmax/Lumora/internal/auth/transport/http"
	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
)

// fakeRepository — in-memory реализация auth/domain.Repository, тот же паттерн,
// что и в internal/auth/service/service_test.go (unit-тесты сервиса), но здесь
// нужна отдельно для HTTP-уровня, так как test-пакеты разные.
type fakeRepository struct {
	mu            sync.Mutex
	usersByID     map[string]domain.User
	usersByEmail  map[string]string
	refreshTokens map[string]domain.RefreshToken
	nextID        int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		usersByID:     map[string]domain.User{},
		usersByEmail:  map[string]string{},
		refreshTokens: map[string]domain.RefreshToken{},
	}
}

func (f *fakeRepository) genID() string {
	f.nextID++
	return fmt.Sprintf("id-%d", f.nextID)
}

func (f *fakeRepository) CreateUser(_ context.Context, email, passwordHash string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.usersByEmail[email]; ok {
		return domain.User{}, domain.ErrEmailTaken
	}

	u := domain.User{ID: f.genID(), Email: email, PasswordHash: passwordHash, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.usersByID[u.ID] = u
	f.usersByEmail[email] = u.ID
	return u, nil
}

func (f *fakeRepository) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	id, ok := f.usersByEmail[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return f.usersByID[id], nil
}

func (f *fakeRepository) GetUserByID(_ context.Context, id string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	u, ok := f.usersByID[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeRepository) CreateRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rt := domain.RefreshToken{ID: f.genID(), UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt, CreatedAt: time.Now()}
	f.refreshTokens[tokenHash] = rt
	return rt, nil
}

func (f *fakeRepository) GetRefreshTokenByHash(_ context.Context, tokenHash string) (domain.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	rt, ok := f.refreshTokens[tokenHash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrRefreshTokenInvalid
	}
	return rt, nil
}

func (f *fakeRepository) RevokeRefreshToken(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for hash, rt := range f.refreshTokens {
		if rt.ID == id {
			now := time.Now()
			rt.RevokedAt = &now
			f.refreshTokens[hash] = rt
			return nil
		}
	}
	return nil
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T) (*httptest.Server, *jwtauth.Issuer) {
	t.Helper()

	issuer := jwtauth.NewIssuer("test-secret", time.Hour)
	svc := service.New(newFakeRepository(), issuer, 720*time.Hour)
	handler := authhttp.NewHandler(svc, noopLogger())

	r := chi.NewRouter()
	authhttp.RegisterRoutes(r, handler, issuer.Middleware)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, issuer
}

func doJSON(t *testing.T, method, url string, body any, bearer string) (*http.Response, map[string]any) {
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

	var decoded map[string]any
	// jwtauth.Issuer.Middleware rejects unauthenticated requests with a plain
	// text 401 (http.Error), not the domain's JSON error shape — only decode
	// when the handler actually produced a JSON body.
	if resp.ContentLength != 0 && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
	}
	return resp, decoded
}

func TestRegisterCreatesUserAndReturnsTokenPair(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email": "ada@example.com", "password": "supersecret1",
	}, "")

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, body["access_token"])
	require.NotEmpty(t, body["refresh_token"])
	user := body["user"].(map[string]any)
	require.Equal(t, "ada@example.com", user["email"])
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	srv, _ := newTestServer(t)

	creds := map[string]string{"email": "ada@example.com", "password": "supersecret1"}
	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/register", creds, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, body := doJSON(t, http.MethodPost, srv.URL+"/auth/register", creds, "")
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	require.Equal(t, domain.ErrEmailTaken.Error(), body["error"])
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, body := doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email": "ada@example.com", "password": "short",
	}, "")

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Equal(t, domain.ErrWeakPassword.Error(), body["error"])
}

func TestRegisterRejectsInvalidBody(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestLoginWithWrongPasswordReturnsUnauthorized(t *testing.T) {
	srv, _ := newTestServer(t)

	doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email": "ada@example.com", "password": "supersecret1",
	}, "")

	resp, body := doJSON(t, http.MethodPost, srv.URL+"/auth/login", map[string]string{
		"email": "ada@example.com", "password": "wrong-password",
	}, "")

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, domain.ErrInvalidCredentials.Error(), body["error"])
}

func TestRefreshRotatesTokenAndInvalidatesThePrevious(t *testing.T) {
	srv, _ := newTestServer(t)

	_, reg := doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email": "ada@example.com", "password": "supersecret1",
	}, "")
	refreshToken := reg["refresh_token"].(string)

	resp, body := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", map[string]string{"refresh_token": refreshToken}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NotEqual(t, refreshToken, body["refresh_token"])

	resp, _ = doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", map[string]string{"refresh_token": refreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "rotated-away refresh token must not be reusable")
}

func TestLogoutIsIdempotent(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/logout", map[string]string{"refresh_token": "unknown-token"}, "")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestMeRequiresBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMeReturnsCurrentUser(t *testing.T) {
	srv, _ := newTestServer(t)

	_, reg := doJSON(t, http.MethodPost, srv.URL+"/auth/register", map[string]string{
		"email": "ada@example.com", "password": "supersecret1",
	}, "")
	accessToken := reg["access_token"].(string)

	resp, body := doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, accessToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "ada@example.com", body["email"])
}

func TestMeRejectsTokenSignedWithDifferentSecret(t *testing.T) {
	srv, _ := newTestServer(t)

	otherIssuer := jwtauth.NewIssuer("a-different-secret", time.Hour)
	token, _, err := otherIssuer.IssueAccessToken("some-user-id")
	require.NoError(t, err)

	resp, _ := doJSON(t, http.MethodGet, srv.URL+"/auth/me", nil, token)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
