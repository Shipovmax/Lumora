package authhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes монтирует эндпоинты домена auth на переданный роутер.
// authMiddleware защищает приватные роуты (например, /me).
func RegisterRoutes(r chi.Router, h *Handler, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)

		r.With(authMiddleware).Get("/me", h.Me)
	})
}
