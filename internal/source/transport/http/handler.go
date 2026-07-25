// Package sourcehttp — HTTP-транспорт домена source: handlers, DTO, роуты.
package sourcehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
	"github.com/Shipovmax/Lumora/internal/source/domain"
	"github.com/Shipovmax/Lumora/internal/source/service"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	var req createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	source, err := h.svc.AddSource(r.Context(), userID, domain.Type(req.Type), req.Name, req.URL)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusCreated, newSourceResponse(source))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	sources, err := h.svc.ListSources(r.Context(), userID)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newSourceListResponse(sources))
}

func (h *Handler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	var req setEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	source, err := h.svc.SetEnabled(r.Context(), userID, chi.URLParam(r, "id"), req.Enabled)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newSourceResponse(source))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	if err := h.svc.DeleteSource(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		writeError(w, h.log, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
