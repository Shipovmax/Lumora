package notificationhttp

import (
	"time"

	"github.com/Shipovmax/Lumora/internal/notification/domain"
)

type registerDeviceRequest struct {
	Platform string `json:"platform"`
	Token    string `json:"token"`
}

type removeDeviceRequest struct {
	Token string `json:"token"`
}

type deviceTokenResponse struct {
	ID        string    `json:"id"`
	Platform  string    `json:"platform"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newDeviceTokenResponse(d domain.DeviceToken) deviceTokenResponse {
	return deviceTokenResponse{
		ID:        d.ID,
		Platform:  d.Platform,
		Token:     d.Token,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
