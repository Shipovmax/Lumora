package userhttp

import (
	"time"

	"github.com/Shipovmax/Lumora/internal/user/domain"
)

type updateProfileRequest struct {
	Name       string   `json:"name"`
	Country    string   `json:"country"`
	Language   string   `json:"language"`
	Profession string   `json:"profession"`
	Interests  []string `json:"interests"`
	Topics     []string `json:"topics"`
}

func (req updateProfileRequest) toDomain() domain.ProfileUpdate {
	return domain.ProfileUpdate{
		Name:       req.Name,
		Country:    req.Country,
		Language:   req.Language,
		Profession: req.Profession,
		Interests:  req.Interests,
		Topics:     req.Topics,
	}
}

type profileResponse struct {
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Country    string    `json:"country"`
	Language   string    `json:"language"`
	Profession string    `json:"profession"`
	Interests  []string  `json:"interests"`
	Topics     []string  `json:"topics"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func newProfileResponse(p domain.Profile) profileResponse {
	return profileResponse{
		UserID:     p.UserID,
		Name:       p.Name,
		Country:    p.Country,
		Language:   p.Language,
		Profession: p.Profession,
		Interests:  p.Interests,
		Topics:     p.Topics,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}
