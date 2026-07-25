package userhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/user/domain"
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
	case errors.Is(err, domain.ErrProfileNotFound):
		status, message = http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrNameTooLong):
		status, message = http.StatusBadRequest, err.Error()
	default:
		log.Error("user: internal error", slog.Any("error", err))
	}

	writeJSON(w, status, map[string]string{"error": message})
}
