package notificationhttp_test

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

	"github.com/Shipovmax/Lumora/internal/notification/domain"
	"github.com/Shipovmax/Lumora/internal/notification/service"
	notificationhttp "github.com/Shipovmax/Lumora/internal/notification/transport/http"
	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
)

type fakeRepository struct {
	mu     sync.Mutex
	tokens map[string]domain.DeviceToken // token -> record
	nextID int
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{tokens: map[string]domain.DeviceToken{}}
}

func (f *fakeRepository) genID() string {
	f.nextID++
	return fmt.Sprintf("device-%d", f.nextID)
}

func (f *fakeRepository) RegisterDevice(_ context.Context, userID, platform, token string) (domain.DeviceToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	dt := domain.DeviceToken{ID: f.genID(), UserID: userID, Platform: platform, Token: token, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.tokens[token] = dt
	return dt, nil
}

func (f *fakeRepository) ListDeviceTokens(_ context.Context, userID string) ([]domain.DeviceToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []domain.DeviceToken
	for _, dt := range f.tokens {
		if dt.UserID == userID {
			out = append(out, dt)
		}
	}
	return out, nil
}

func (f *fakeRepository) RemoveDeviceToken(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.tokens, token)
	return nil
}

// fakeSender — never invoked from HTTP handlers (only NotifyEvent from the
// worker uses it), but service.New requires one.
type fakeSender struct{}

func (fakeSender) Send(context.Context, string, domain.PushMessage) error { return nil }

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(t *testing.T) (*httptest.Server, *jwtauth.Issuer) {
	t.Helper()

	issuer := jwtauth.NewIssuer("test-secret", time.Hour)
	svc := service.New(newFakeRepository(), fakeSender{}, noopLogger())
	handler := notificationhttp.NewHandler(svc, noopLogger())

	r := chi.NewRouter()
	notificationhttp.RegisterRoutes(r, handler, issuer.Middleware)

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

func TestRegisterDeviceRequiresBearerToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", "", map[string]string{"platform": "android", "token": "t"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRegisterDeviceSuccess(t *testing.T) {
	srv, issuer := newTestServer(t)
	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", token, map[string]string{
		"platform": "android", "token": "fcm-token-abc",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, "android", body["platform"])
	require.Equal(t, "fcm-token-abc", body["token"])
}

func TestRegisterDeviceRejectsInvalidPlatform(t *testing.T) {
	srv, issuer := newTestServer(t)
	accessToken, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", accessToken, map[string]string{
		"platform": "windows-phone", "token": "t",
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrInvalidPlatform.Error(), body["error"])
}

func TestRegisterDeviceRejectsMissingToken(t *testing.T) {
	srv, issuer := newTestServer(t)
	accessToken, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", accessToken, map[string]string{
		"platform": "android", "token": "",
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrTokenRequired.Error(), body["error"])
}

func TestRemoveDeviceRejectsTokenNotOwnedByCaller(t *testing.T) {
	srv, issuer := newTestServer(t)
	tokenA, _, err := issuer.IssueAccessToken("user-a")
	require.NoError(t, err)
	tokenB, _, err := issuer.IssueAccessToken("user-b")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", tokenA, map[string]string{"platform": "ios", "token": "device-a"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp = do(t, http.MethodDelete, srv.URL+"/notifications/devices/", tokenB, map[string]string{"token": "device-a"})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	body := decode(t, resp)
	require.Equal(t, domain.ErrInvalidToken.Error(), body["error"])
}

func TestRemoveDeviceSuccess(t *testing.T) {
	srv, issuer := newTestServer(t)
	accessToken, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	resp := do(t, http.MethodPost, srv.URL+"/notifications/devices/", accessToken, map[string]string{"platform": "ios", "token": "device-1"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp = do(t, http.MethodDelete, srv.URL+"/notifications/devices/", accessToken, map[string]string{"token": "device-1"})
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}
