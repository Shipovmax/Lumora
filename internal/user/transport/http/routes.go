package userhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes монтирует эндпоинты домена user на переданный роутер.
// Все роуты домена приватные — authMiddleware обязателен для всей группы.
func RegisterRoutes(r chi.Router, h *Handler, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/user", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/profile", h.GetProfile)
		r.Put("/profile", h.UpdateProfile)
	})
}
