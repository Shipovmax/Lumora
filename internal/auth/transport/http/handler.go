// Package authhttp — HTTP-транспорт домена auth: handlers, DTO, роуты.
package authhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/auth/service"
	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusCreated, newAuthResponse(result))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newAuthResponse(result))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	result, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newAuthResponse(result))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		writeError(w, h.log, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}
