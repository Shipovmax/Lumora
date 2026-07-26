package jwtauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
)

func TestIssueAndParseAccessTokenRoundTrip(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	token, expiresAt, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, time.Second)

	userID, err := issuer.ParseAccessToken(token)
	require.NoError(t, err)
	require.Equal(t, "user-1", userID)
}

func TestParseAccessTokenRejectsExpiredToken(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", -time.Hour)

	token, _, err := issuer.IssueAccessToken("user-1")
	require.NoError(t, err)

	_, err = issuer.ParseAccessToken(token)
	require.ErrorIs(t, err, jwtauth.ErrInvalidToken)
}

func TestParseAccessTokenRejectsWrongSecret(t *testing.T) {
	issuedBy := jwtauth.NewIssuer("secret-a", time.Hour)
	verifiedBy := jwtauth.NewIssuer("secret-b", time.Hour)

	token, _, err := issuedBy.IssueAccessToken("user-1")
	require.NoError(t, err)

	_, err = verifiedBy.ParseAccessToken(token)
	require.ErrorIs(t, err, jwtauth.ErrInvalidToken)
}

func TestParseAccessTokenRejectsMalformedToken(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	_, err := issuer.ParseAccessToken("not-a-jwt")
	require.ErrorIs(t, err, jwtauth.ErrInvalidToken)
}

// TestParseAccessTokenRejectsNoneAlgorithm гарантирует защиту от классической
// атаки "alg confusion": токен, подписанный (точнее, не подписанный) с
// alg=none, не должен приниматься, даже если Subject валиден.
func TestParseAccessTokenRejectsNoneAlgorithm(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   "user-1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})
	token, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = issuer.ParseAccessToken(token)
	require.ErrorIs(t, err, jwtauth.ErrInvalidToken)
}

func TestMiddlewareRejectsMissingAuthorizationHeader(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	issuer.Middleware(neverCalledHandler(t)).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareRejectsMalformedAuthorizationHeader(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	issuer.Middleware(neverCalledHandler(t)).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer garbage")

	issuer.Middleware(neverCalledHandler(t)).ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMiddlewarePutsUserIDInContext(t *testing.T) {
	issuer := jwtauth.NewIssuer("test-secret", time.Hour)

	token, _, err := issuer.IssueAccessToken("user-42")
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	var gotUserID string
	var gotOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, gotOK = jwtauth.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	issuer.Middleware(next).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, gotOK)
	require.Equal(t, "user-42", gotUserID)
}

func TestUserIDFromContextMissing(t *testing.T) {
	_, ok := jwtauth.UserIDFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context())
	require.False(t, ok)
}

func neverCalledHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called for a rejected request")
	})
}
