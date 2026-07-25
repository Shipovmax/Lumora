// Package jwtauth выпускает и валидирует короткоживущие access-JWT и
// предоставляет chi-middleware, извлекающий идентификатор пользователя
// из заголовка Authorization. Refresh-токены — не JWT, они выпускаются
// и хранятся доменом auth (см. internal/auth) для возможности отзыва.
package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

type claims struct {
	jwt.RegisteredClaims
}

// IssueAccessToken выпускает подписанный JWT с subject = userID.
func (i *Issuer) IssueAccessToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(i.ttl)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})

	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

// ParseAccessToken проверяет подпись и срок действия токена, возвращая userID (subject).
func (i *Issuer) ParseAccessToken(raw string) (string, error) {
	token, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	c, ok := token.Claims.(*claims)
	if !ok || c.Subject == "" {
		return "", ErrInvalidToken
	}

	return c.Subject, nil
}

type contextKey string

const userIDContextKey contextKey = "user_id"

// Middleware проверяет заголовок "Authorization: Bearer <token>" и кладёт
// userID в контекст запроса. При отсутствии/невалидности токена отвечает 401.
func (i *Issuer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}

		userID, err := i.ParseAccessToken(token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext возвращает userID, положенный Middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}
