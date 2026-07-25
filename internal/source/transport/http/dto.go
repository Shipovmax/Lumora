package sourcehttp

import (
	"time"

	"github.com/Shipovmax/Lumora/internal/source/domain"
)

type createSourceRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type setEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type sourceResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newSourceResponse(s domain.Source) sourceResponse {
	return sourceResponse{
		ID:        s.ID,
		Type:      string(s.Type),
		Name:      s.Name,
		URL:       s.URL,
		Enabled:   s.Enabled,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func newSourceListResponse(sources []domain.Source) []sourceResponse {
	out := make([]sourceResponse, 0, len(sources))
	for _, s := range sources {
		out = append(out, newSourceResponse(s))
	}
	return out
}
