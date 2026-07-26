package notificationhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes монтирует эндпоинты домена notification на переданный
// роутер. Все роуты приватные — authMiddleware обязателен для всей группы.
func RegisterRoutes(r chi.Router, h *Handler, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/notifications/devices", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.RegisterDevice)
		r.Delete("/", h.RemoveDevice)
	})
}
