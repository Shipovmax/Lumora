// Package usercontexthttp — HTTP-транспорт домена usercontext: handlers, DTO, роуты.
package usercontexthttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/usercontext/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) GetContext(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	c, err := h.svc.GetContext(r.Context(), userID)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newContextResponse(c))
}

func (h *Handler) UpdateContext(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	var req updateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	c, err := h.svc.UpdateContext(r.Context(), userID, req.Content)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newContextResponse(c))
}
