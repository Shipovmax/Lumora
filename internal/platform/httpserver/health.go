package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
)

// Checker — проверка доступности одной зависимости (Postgres, Redis, ...) для /readyz.
type Checker func(ctx context.Context) error

// RegisterHealth регистрирует:
//   - /healthz — liveness: процесс жив, отвечает всегда 200.
//   - /readyz  — readiness: 200, только если все переданные зависимости доступны.
func RegisterHealth(mux interface {
	Get(pattern string, h http.HandlerFunc)
}, checks map[string]Checker) {
	mux.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK
		result := make(map[string]string, len(checks))

		for name, check := range checks {
			if err := check(r.Context()); err != nil {
				status = http.StatusServiceUnavailable
				result[name] = err.Error()
				continue
			}
			result[name] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
	})
}
