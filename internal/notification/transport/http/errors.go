package notificationhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
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
	case errors.Is(err, domain.ErrTokenRequired), errors.Is(err, domain.ErrInvalidPlatform):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrInvalidToken):
		status, message = http.StatusNotFound, err.Error()
	default:
		log.Error("notification: internal error", slog.Any("error", err))
	}

	writeJSON(w, status, map[string]string{"error": message})
}
