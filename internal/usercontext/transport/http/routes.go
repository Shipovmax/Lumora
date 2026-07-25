package usercontexthttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes монтирует эндпоинты домена usercontext на переданный роутер.
// Все роуты домена приватные — authMiddleware обязателен для всей группы.
func RegisterRoutes(r chi.Router, h *Handler, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/context", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/", h.GetContext)
		r.Put("/", h.UpdateContext)
	})
}
