package usercontexthttp

import (
	"time"

	"github.com/Shipovmax/Lumora/internal/usercontext/domain"
)

type updateContextRequest struct {
	Content string `json:"content"`
}

type contextResponse struct {
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newContextResponse(c domain.Context) contextResponse {
	return contextResponse{
		UserID:    c.UserID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
