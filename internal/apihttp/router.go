// Package apihttp собирает transport/http всех доменов в единый REST API
// под префиксом /api/v1 (Этап 11 дополнит его OpenAPI-документацией).
package apihttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	authhttp "github.com/Shipovmax/Lumora/internal/auth/transport/http"
	userhttp "github.com/Shipovmax/Lumora/internal/user/transport/http"
)

type Deps struct {
	Auth           *authhttp.Handler
	User           *userhttp.Handler
	AuthMiddleware func(http.Handler) http.Handler
}

// Mount регистрирует доменные роуты под /api/v1 на переданном роутере.
func Mount(r chi.Router, deps Deps) {
	r.Route("/api/v1", func(r chi.Router) {
		authhttp.RegisterRoutes(r, deps.Auth, deps.AuthMiddleware)
		userhttp.RegisterRoutes(r, deps.User, deps.AuthMiddleware)
	})
}
