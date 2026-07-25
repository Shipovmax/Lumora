package sourcehttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes монтирует эндпоинты домена source на переданный роутер.
// Все роуты домена приватные — authMiddleware обязателен для всей группы.
func RegisterRoutes(r chi.Router, h *Handler, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/sources", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Patch("/{id}", h.SetEnabled)
		r.Delete("/{id}", h.Delete)
	})
}
