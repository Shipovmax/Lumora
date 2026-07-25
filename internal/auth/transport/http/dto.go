package authhttp

import (
	"time"

	"github.com/Shipovmax/Lumora/internal/auth/domain"
	"github.com/Shipovmax/Lumora/internal/auth/service"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type userResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type authResponse struct {
	AccessToken          string       `json:"access_token"`
	AccessTokenExpiresAt time.Time    `json:"access_token_expires_at"`
	RefreshToken         string       `json:"refresh_token"`
	User                 userResponse `json:"user"`
}

func newAuthResponse(r service.AuthResult) authResponse {
	return authResponse{
		AccessToken:          r.AccessToken,
		AccessTokenExpiresAt: r.AccessTokenExpiresAt,
		RefreshToken:         r.RefreshToken,
		User:                 newUserResponse(r.User),
	}
}

func newUserResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, CreatedAt: u.CreatedAt}
}
