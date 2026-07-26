package sourcehttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Shipovmax/Lumora/internal/source/domain"
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
	case errors.Is(err, domain.ErrSourceNotFound):
		status, message = http.StatusNotFound, err.Error()
	case errors.Is(err, domain.ErrInvalidType), errors.Is(err, domain.ErrNameRequired), errors.Is(err, domain.ErrURLRequired),
		errors.Is(err, domain.ErrUnsupportedURLScheme):
		status, message = http.StatusBadRequest, err.Error()
	default:
		log.Error("source: internal error", slog.Any("error", err))
	}

	writeJSON(w, status, map[string]string{"error": message})
}
