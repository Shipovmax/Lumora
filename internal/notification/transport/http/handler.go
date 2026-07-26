// Package notificationhttp — HTTP-транспорт домена notification: регистрация
// и удаление токенов устройств для push-уведомлений (сама отправка — из
// воркера, см. internal/notification/transport/worker).
package notificationhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/notification/service"
	"github.com/Shipovmax/Lumora/internal/platform/jwtauth"
)

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func NewHandler(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	var req registerDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	device, err := h.svc.RegisterDevice(r.Context(), userID, req.Platform, req.Token)
	if err != nil {
		writeError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, newDeviceTokenResponse(device))
}

func (h *Handler) RemoveDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := jwtauth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}

	var req removeDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.svc.RemoveDeviceToken(r.Context(), userID, req.Token); err != nil {
		writeError(w, h.log, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
