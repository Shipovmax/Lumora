package authhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, log *slog.Logger, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		status, message = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrInvalidCredentials):
		status, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, domain.ErrRefreshTokenInvalid):
		status, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrWeakPassword):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUserNotFound):
		status, message = http.StatusNotFound, err.Error()
	default:
		log.Error("auth: internal error", slog.Any("error", err))
	}

	writeJSON(w, status, map[string]string{"error": message})
}
